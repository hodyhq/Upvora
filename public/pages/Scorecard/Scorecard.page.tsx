import React from "react"
import { Button, Header } from "@fider/components"
import { HStack, VStack } from "@fider/components/layout"
import { actions, Fider, notify } from "@fider/services"

interface ScorecardRecord {
  id: number
  postId?: number
  title: string
  values: unknown
  createdAt: string
  updatedAt: string
}

interface ScorecardPageProps {
  scorecards?: ScorecardRecord[]
}

const Scorecard: React.FC<ScorecardPageProps> = (props) => {
  const cards = props.scorecards ?? []
  const featureOff = !Fider.session.tenant.isScorecardEnabled

  return (
    <>
      <Header />
      <div id="p-scorecard" className="page container">
        <VStack spacing={4}>
          <HStack className="justify-between items-baseline">
            <div>
              <h1 className="text-header">Scorecard</h1>
              <p className="text-muted">Committee-scored review of ideas — weighted across the scoring dimensions defined in Admin → Scorecard Fields.</p>
            </div>
            <Button
              variant="primary"
              onClick={async () => {
                const r = await actions.createScorecard({})
                if (r.ok && r.data) {
                  window.location.href = `/scorecard/${r.data.id}`
                } else {
                  notify.error("Could not create scorecard.")
                }
              }}
            >
              + New scorecard
            </Button>
          </HStack>

          {featureOff && (
            <div className="p-3 rounded bg-yellow-100 text-yellow-800">
              Scorecard feature is <strong>disabled</strong> at the tenant level. An administrator can turn it on at <a href="/admin/scorecard-settings" className="text-link">Site Settings → Scorecard</a>.
            </div>
          )}

          {cards.length === 0 ? (
            <div className="p-6 text-center bg-white rounded shadow">
              <p className="text-muted">No scorecards yet.</p>
              <p className="text-muted text-sm mt-2">
                Scorecards get created either automatically (when a post's status changes to one flagged as a scorecard trigger) or manually from a post's detail page (button coming in a later build).
              </p>
            </div>
          ) : (
            <table className="w-full bg-white rounded shadow">
              <thead>
                <tr className="text-left text-xs uppercase text-muted">
                  <th className="p-3">Title</th>
                  <th className="p-3">Post</th>
                  <th className="p-3">Updated</th>
                </tr>
              </thead>
              <tbody>
                {cards.map((c) => (
                  <tr key={c.id} className="border-t">
                    <td className="p-3">
                      <a href={`/scorecard/${c.id}`} className="text-link">
                        {c.title || `Scorecard #${c.id}`}
                      </a>
                    </td>
                    <td className="p-3 text-sm text-muted">{c.postId != null ? `#${c.postId}` : "—"}</td>
                    <td className="p-3 text-sm text-muted">{new Date(c.updatedAt).toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </VStack>
      </div>
    </>
  )
}

export default Scorecard
