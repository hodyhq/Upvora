import React from "react"

import { Modal, Button, DisplayError, Select, Form, TextArea, Field, SelectOption } from "@fider/components"
import { Post, PostStatus, statusListFor } from "@fider/models"

import { actions, Failure, Fider } from "@fider/services"
import { PostSearch } from "./PostSearch"
import { i18n } from "@lingui/core"
import { Trans } from "@lingui/react/macro"

interface ResponseModalProps {
  post: Post
  showModal: boolean
  onCloseModal: () => void
}

interface ResponseModalState {
  status: string
  text: string
  originalNumber: number
  error?: Failure
}

export class ResponseModal extends React.Component<ResponseModalProps, ResponseModalState> {
  constructor(props: ResponseModalProps) {
    super(props)

    this.state = {
      status: this.props.post.status,
      originalNumber: 0,
      text: this.props.post.response ? this.props.post.response.text : "",
    }
  }

  private submit = async () => {
    const result = await actions.respond(this.props.post.number, this.state)
    if (result.ok) {
      location.reload()
    } else {
      this.setState({
        error: result.error,
      })
    }
  }

  private setStatus = (opt?: SelectOption) => {
    if (opt) {
      this.setState({ status: opt.value })
    }
  }

  private setOriginalNumber = (originalNumber: number) => {
    this.setState({ originalNumber })
  }

  private setText = (text: string) => {
    this.setState({ text })
  }

  public render() {
    // Prefer the tenant's configured status catalogue (feedback.fider.io/111).
    // Falls back to the static PostStatus.All for tenants whose statuses haven't
    // been seeded yet (e.g. fresh installs that haven't run the migration).
    const statusList = statusListFor(Fider.session.tenant)
    const options = statusList.map((s) => {
      const id = `enum.poststatus.${s.value}`
      return {
        value: s.value,
        // Translation key wins when present; falls back to the tenant-defined
        // label (which is what custom statuses use).
        label: i18n._(id, { message: s.label }),
      }
    })

    const modal = (
      <Modal.Window isOpen={this.props.showModal} onClose={this.props.onCloseModal} center={false} size="large">
        <Modal.Content>
          <Form error={this.state.error} className="c-response-form">
            <Select field="status" label="Status" defaultValue={this.state.status} options={options} onChange={this.setStatus} />
            {this.state.status === PostStatus.Duplicate.value ? (
              <>
                <Field>
                  <PostSearch exclude={[this.props.post.number]} onChanged={this.setOriginalNumber} />
                </Field>
                <DisplayError fields={["originalNumber"]} error={this.state.error} />
                <span className="text-muted">
                  <Trans id="showpost.responseform.message.mergedvotes">Votes from this post will be merged into original post.</Trans>
                </span>
              </>
            ) : (
              <TextArea
                field="text"
                onChange={this.setText}
                value={this.state.text}
                minRows={5}
                placeholder={i18n._({
                  id: "showpost.responseform.text.placeholder",
                  message: "What's going on with this post? Let your users know what are your plans...",
                })}
              />
            )}
          </Form>
        </Modal.Content>

        <Modal.Footer>
          <Button variant="primary" onClick={this.submit}>
            <Trans id="action.submit">Submit</Trans>
          </Button>
          <Button variant="tertiary" onClick={this.props.onCloseModal}>
            <Trans id="action.cancel">Cancel</Trans>
          </Button>
        </Modal.Footer>
      </Modal.Window>
    )

    return modal
  }
}
