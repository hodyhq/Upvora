import React from "react"
import { Button, Header } from "@fider/components"
import { HStack, VStack } from "@fider/components/layout"
import { actions, Fider, notify } from "@fider/services"
import { ScoreBandPill, computeWeightedScore } from "./ScoreBand"

interface ScorecardRecord {
  id: number
  postId?: number
  title: string
  values: Record<string, unknown>
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
            <div style={{ background: "var(--colors-white)", border: "1px solid var(--colors-gray-200)", borderRadius: 10, padding: 32, textAlign: "center" }}>
              <p className="text-muted">No scorecards yet.</p>
              <p className="text-muted text-sm mt-2">
                Cards get created automatically when a post's status changes to your configured trigger, or manually from the "Score this idea" button on a post's detail page.
              </p>
            </div>
          ) : (
            <div style={{ background: "var(--colors-white)", border: "1px solid var(--colors-gray-200)", borderRadius: 10, overflow: "hidden" }}>
              <table style={{ width: "100%", borderCollapse: "collapse" }}>
                <thead>
                  <tr style={{ background: "var(--colors-gray-50, #f8fafc)", borderBottom: "1px solid var(--colors-gray-200)" }}>
                    <th style={{ textAlign: "left", padding: "10px 14px", fontSize: 11, textTransform: "uppercase", letterSpacing: "0.06em", color: "var(--colors-gray-500)", fontWeight: 600 }}>Title</th>
                    <th style={{ textAlign: "left", padding: "10px 14px", fontSize: 11, textTransform: "uppercase", letterSpacing: "0.06em", color: "var(--colors-gray-500)", fontWeight: 600 }}>Score</th>
                    <th style={{ textAlign: "left", padding: "10px 14px", fontSize: 11, textTransform: "uppercase", letterSpacing: "0.06em", color: "var(--colors-gray-500)", fontWeight: 600 }}>Post</th>
                    <th style={{ textAlign: "left", padding: "10px 14px", fontSize: 11, textTransform: "uppercase", letterSpacing: "0.06em", color: "var(--colors-gray-500)", fontWeight: 600 }}>Updated</th>
                  </tr>
                </thead>
                <tbody>
                  {cards.map((c, i) => {
                    const score = computeWeightedScore(c.values, Fider.session.tenant.scorecardFields)
                    return (
                      <tr key={c.id} style={{ borderTop: i === 0 ? "none" : "1px solid var(--colors-gray-100)" }}>
                        <td style={{ padding: "12px 14px" }}>
                          <a href={`/scorecard/${c.id}`} className="text-link" style={{ fontWeight: 500 }}>
                            {c.title || `Scorecard #${c.id}`}
                          </a>
                        </td>
                        <td style={{ padding: "12px 14px" }}>
                          <ScoreBandPill score={score} />
                        </td>
                        <td style={{ padding: "12px 14px", fontSize: 13, color: "var(--colors-gray-500)" }}>
                          {c.postId != null ? (
                            <a href={`/posts/${c.postId}`} className="text-link" style={{ fontSize: 13 }}>#{c.postId}</a>
                          ) : "—"}
                        </td>
                        <td style={{ padding: "12px 14px", fontSize: 13, color: "var(--colors-gray-500)" }}>{new Date(c.updatedAt).toLocaleString()}</td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          )}
        </VStack>
      </div>
    </>
  )
}

export default Scorecard
