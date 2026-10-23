package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/dbx"
	"github.com/getfider/fider/app/pkg/errors"
	"github.com/getfider/fider/app/pkg/rand"
	"github.com/getfider/fider/app/pkg/validate"
	"github.com/getfider/fider/app/services/blob"
)

func setAttachments(ctx context.Context, c *cmd.SetAttachments) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		postID := c.Post.ID
		var commentID sql.NullInt64
		if c.Comment != nil {
			err := commentID.Scan(c.Comment.ID)
			if err != nil {
				return errors.Wrap(err, "failed scan comment id")
			}
		}

		for _, attachment := range c.Attachments {
			if attachment.Remove {
				if _, err := trx.Execute(
					"DELETE FROM attachments WHERE tenant_id = $1 AND post_id = $2 AND (comment_id = $3 OR ($3 IS NULL AND comment_id IS NULL)) AND attachment_bkey = $4",
					tenant.ID, postID, commentID, attachment.BlobKey,
				); err != nil {
					return errors.Wrap(err, "failed to delete attachment")
				}
			} else if attachment.Upload != nil && len(attachment.Upload.Content) > 0 {
				// Only fresh uploads become attachments: an entry naming a key alone
				// keeps what this idea already has and must never attach another
				// idea's file (access to a file follows the idea it is attached to).
				// Use a conditional INSERT that only adds the attachment if it doesn't already exist
				if _, err := trx.Execute(`
					INSERT INTO attachments (tenant_id, post_id, comment_id, user_id, attachment_bkey)
					SELECT $1, $2, $3, $4, $5
					WHERE NOT EXISTS (
						SELECT 1 FROM attachments
						WHERE tenant_id = $6
						AND post_id = $7
						AND (comment_id = $8 OR ($8 IS NULL AND comment_id IS NULL))
						AND attachment_bkey = $9
					)
				`, tenant.ID, postID, commentID, user.ID, attachment.BlobKey,
					tenant.ID, postID, commentID, attachment.BlobKey); err != nil {
					return errors.Wrap(err, "failed to insert attachment")
				}
			}
		}

		return nil
	})
}

func getAttachments(ctx context.Context, q *query.GetAttachments) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		q.Result = make([]string, 0)

		postID := q.Post.ID
		var commentID sql.NullInt64
		if q.Comment != nil {
			err := commentID.Scan(q.Comment.ID)
			if err != nil {
				return errors.Wrap(err, "failed scan comment id")
			}
		}

		type entry struct {
			BlobKey string `db:"attachment_bkey"`
		}

		entries := []*entry{}
		err := trx.Select(&entries, `
			SELECT attachment_bkey
			FROM attachments
			WHERE tenant_id = $1 AND post_id = $2 AND (comment_id = $3 OR ($3 IS NULL AND comment_id IS NULL))
		`, tenant.ID, postID, commentID)
		if err != nil {
			return errors.Wrap(err, "failed to get attachments")
		}

		q.Result = make([]string, len(entries))
		for i, entry := range entries {
			q.Result[i] = entry.BlobKey
		}

		return nil
	})
}

func uploadImage(ctx context.Context, c *cmd.UploadImage) error {
	if c.Image.Upload != nil && len(c.Image.Upload.Content) > 0 {
		// Never trust the client-supplied content type, it is served back as-is
		folder := c.Folder
		// Files are recognised first, exactly as validate.MultiImageUpload does,
		// so an upload is stored as what it was validated as.
		if contentType, ext, isFile := validate.DetectFile(c.Image.Upload.Content, c.Image.Upload.FileName); c.AllowFiles && isFile {
			// Kept apart from images so the image route never serves it and
			// /static/files can check access.
			folder = "files"
			c.Image.Upload.ContentType = contentType
			name := c.Image.Upload.FileName
			c.Image.Upload.FileName = strings.TrimSuffix(name, filepath.Ext(name)) + ext
		} else {
			header, err := validate.ReadImageHeader(c.Image.Upload.Content)
			if err != nil {
				return errors.Wrap(err, "failed to read image header")
			}
			c.Image.Upload.ContentType = header.ContentType()
		}

		// Regenerate key if empty or wrong folder prefix (prevents cross-folder
		// overwrites); file keys are always ours.
		if folder == "files" || c.Image.BlobKey == "" || !strings.HasPrefix(c.Image.BlobKey, folder+"/") {
			c.Image.BlobKey = fmt.Sprintf("%s/%s-%s", folder, rand.String(64), blob.SanitizeFileName(c.Image.Upload.FileName))
		}

		// If a blob already exists at this key, generate a new key to prevent overwriting
		existsQ := &query.GetBlobByKey{Key: c.Image.BlobKey}
		if err := bus.Dispatch(ctx, existsQ); err == nil {
			c.Image.BlobKey = fmt.Sprintf("%s/%s-%s", folder, rand.String(64), blob.SanitizeFileName(c.Image.Upload.FileName))
		}

		err := bus.Dispatch(ctx, &cmd.StoreBlob{
			Key:         c.Image.BlobKey,
			Content:     c.Image.Upload.Content,
			ContentType: c.Image.Upload.ContentType,
		})
		if err != nil {
			return errors.Wrap(err, "failed to upload new blob")
		}
	}
	return nil
}

func uploadImages(ctx context.Context, c *cmd.UploadImages) error {
	for _, img := range c.Images {
		if err := bus.Dispatch(ctx, &cmd.UploadImage{
			Image:      img,
			Folder:     c.Folder,
			AllowFiles: c.AllowFiles,
		}); err != nil {
			return err
		}
	}
	return nil
}

func getPostIDByAttachment(ctx context.Context, q *query.GetPostIDByAttachment) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		err := trx.Scalar(&q.Result, `
			SELECT post_id FROM attachments
			WHERE tenant_id = $1 AND attachment_bkey = $2 AND comment_id IS NULL
			ORDER BY id LIMIT 1`, tenant.ID, q.BlobKey)
		if err != nil {
			return errors.Wrap(err, "failed to find the idea of an attachment")
		}
		return nil
	})
}
