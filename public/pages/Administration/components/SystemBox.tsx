import React, { useEffect, useRef, useState } from "react"
import { actions } from "@fider/services"
import { SystemStatus } from "@fider/services/actions/system"
import { useFider } from "@fider/hooks"

// Pinned under the admin side menu on every settings page: current version,
// latest GitHub release (server caches the check for a day), and — when the
// updater sidecar is configured on the host — a one-click update.
export const SystemBox = () => {
  const fider = useFider()
  const [status, setStatus] = useState<SystemStatus | undefined>()
  const [checking, setChecking] = useState(false)
  const [armed, setArmed] = useState(false)
  const [updating, setUpdating] = useState(false)
  const [updateError, setUpdateError] = useState(false)
  const pollRef = useRef(0)

  const fetchStatus = async (force = false) => {
    const result = await actions.getSystemStatus(force)
    if (result.ok) {
      setStatus(result.data)
    }
    return result
  }

  useEffect(() => {
    fetchStatus()
    return () => window.clearInterval(pollRef.current)
  }, [])

  const checkNow = async () => {
    setChecking(true)
    await fetchStatus(true)
    setChecking(false)
  }

  const startUpdate = async () => {
    if (!armed) {
      setArmed(true)
      return
    }
    setArmed(false)
    const result = await actions.triggerSystemUpdate()
    if (!result.ok) {
      setUpdateError(true)
      return
    }
    setUpdating(true)
    pollRef.current = window.setInterval(async () => {
      try {
        const polled = await actions.getSystemStatus()
        if (!polled.ok) {
          return // app restarting — keep polling
        }
        setStatus(polled.data)
        if (polled.data.currentVersion !== fider.settings.version || polled.data.updateStatus === "done") {
          window.clearInterval(pollRef.current)
          window.location.reload()
        } else if (polled.data.updateStatus === "error") {
          window.clearInterval(pollRef.current)
          setUpdating(false)
          setUpdateError(true)
        }
      } catch {
        // network hiccup while the container restarts — keep polling
      }
    }, 3000)
  }

  const state = status?.state
  const stateLabel =
    state === "up-to-date" ? "Up to date" : state === "update-available" ? "Update available" : state === "ahead" ? "Ahead of latest release" : "—"

  return (
    <div className="c-system-box rounded-md shadow bg-white">
      <span className="c-system-box__eyebrow">System</span>
      <div className="c-system-box__row">
        <span>Installed</span>
        <code>{fider.settings.version}</code>
      </div>
      <div className="c-system-box__row">
        <span>Latest release</span>
        {status?.latest ? (
          <a href={status.latest.url} target="_blank" rel="noopener noreferrer">
            <code>{status.latest.tag}</code>
          </a>
        ) : (
          <code>—</code>
        )}
      </div>
      <div className={`c-system-box__state c-system-box__state--${state ?? "unknown"}`}>{stateLabel}</div>

      {updating ? (
        <div className="c-system-box__updating">
          Updating… the site will restart and this page will reload itself.
          {status?.updateLog && <pre>{status.updateLog.split("\n").slice(-1)[0]}</pre>}
        </div>
      ) : (
        <div className="c-system-box__actions">
          <button className="c-system-box__link" onClick={checkNow} disabled={checking}>
            {checking ? "Checking…" : "Check for updates"}
          </button>
          {status?.updaterConfigured && (
            <button className={armed ? "c-system-box__update c-system-box__update--armed" : "c-system-box__update"} onClick={startUpdate}>
              {armed ? "Click again to confirm" : "Update now"}
            </button>
          )}
        </div>
      )}

      {updateError && <div className="c-system-box__error">The update failed to start or errored — see the recovery steps below.</div>}

      <details className="c-system-box__recovery" open={updateError}>
        <summary>If the site doesn&apos;t come back</summary>
        <p>SSH into the host, go to the compose directory and check the logs:</p>
        <pre>
          docker compose logs app --tail 100{"\n"}
          docker compose logs updater --tail 50
        </pre>
        <p>
          <code>docker compose up -d</code> brings everything back with the images already on disk.
        </p>
      </details>
    </div>
  )
}
