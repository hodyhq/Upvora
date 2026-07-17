import "./VoraChat.scss"

import React, { useEffect, useRef, useState } from "react"
import { AIMessage } from "@fider/models"
import { actions, Fider, notify } from "@fider/services"
import { Markdown } from "@fider/components/common/Markdown"

interface VoraChatProps {
  productId: number
  onDone: (title: string, description: string, brief: string, tags: string[]) => void
  onClose: () => void
}

// Vora — the ideation agent. Pure conversation: the only outputs are text
// and, at the end, a {title, description, brief} the user reviews.
//
// Delivery is resilient by design: a failed turn keeps the user's message in
// the log with an inline Retry (never dumped back into the input), and every
// turn gets one silent retry first — the edge occasionally drops a response
// the server actually produced.
export const VoraChat = (props: VoraChatProps) => {
  const agent = Fider.session.tenant.aiAgents?.find((a) => a.productId === props.productId) ?? Fider.session.tenant.aiAgents?.find((a) => a.productId === null)

  const [messages, setMessages] = useState<AIMessage[]>([])
  const [input, setInput] = useState("")
  const [busy, setBusy] = useState(false)
  const [failed, setFailed] = useState(false)
  const [finalizing, setFinalizing] = useState(false)
  const logRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    logRef.current?.scrollTo({ top: logRef.current.scrollHeight })
  }, [messages, busy, failed])

  const deliver = async (history: AIMessage[]): Promise<{ reply: string; ready: boolean } | null> => {
    try {
      const result = await actions.aiIdeate(props.productId, history)
      if (result.ok) {
        return result.data
      }
    } catch {
      // network-level failure — treated the same as a bad response
    }
    return null
  }

  const run = async (history: AIMessage[]) => {
    setMessages(history)
    setFailed(false)
    setBusy(true)
    let res = await deliver(history)
    if (res === null) {
      await new Promise((resolve) => setTimeout(resolve, 2000))
      res = await deliver(history)
    }
    setBusy(false)
    if (res !== null) {
      const next: AIMessage[] = [...history, { role: "assistant", content: res.reply }]
      setMessages(next)
      if (res.ready) {
        // Vora announced it has enough — draft immediately, no button needed.
        await wrapUp(next)
      }
    } else {
      setFailed(true)
    }
  }

  const send = () => {
    const text = input.trim()
    if (!text || busy || finalizing) {
      return
    }
    setInput("")
    run([...messages, { role: "user", content: text }])
  }

  const retry = () => {
    if (!busy && !finalizing) {
      run(messages)
    }
  }

  const wrapUp = async (history: AIMessage[]) => {
    if (finalizing || history.length < 2) {
      return
    }
    setFinalizing(true)
    try {
      const result = await actions.aiFinalize(props.productId, history)
      if (result.ok) {
        props.onDone(result.data.title, result.data.description, result.data.brief, result.data.tags ?? [])
        return
      }
      notify.error("Vora couldn't wrap up — try once more.")
    } catch {
      notify.error("Vora couldn't wrap up — try once more.")
    } finally {
      setFinalizing(false)
    }
  }

  return (
    <div className="c-vora">
      <div className="c-vora__head">
        <span className="c-vora__name">✦ Vora</span>
        <span className="c-vora__tag">plans the idea with you — you review everything before it posts</span>
        <button type="button" className="c-vora__close" onClick={props.onClose} aria-label="Close Vora">
          ×
        </button>
      </div>

      <div className="c-vora__log" ref={logRef}>
        <div className="c-vora__msg c-vora__msg--agent">
          {agent?.description || "Hey! I'm Vora — tell me your rough idea and I'll help you turn it into something the team can evaluate."}
        </div>
        {messages.map((m, i) => (
          <div key={i} className={`c-vora__msg ${m.role === "assistant" ? "c-vora__msg--agent" : "c-vora__msg--user"}`}>
            {m.role === "assistant" ? <Markdown text={m.content} style="full" /> : m.content}
          </div>
        ))}
        {(busy || finalizing) && <div className="c-vora__thinking">{finalizing ? "Vora is drafting your idea…" : "Vora is thinking…"}</div>}
        {failed && !busy && (
          <div className="c-vora__fail">
            That didn&apos;t get through.
            <button type="button" onClick={retry}>
              Retry
            </button>
          </div>
        )}
      </div>

      <div className="c-vora__box">
        <input
          value={input}
          disabled={busy || finalizing}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              send()
            }
          }}
          placeholder={messages.length === 0 ? "Describe your idea in a sentence or two…" : "Answer Vora…"}
        />
        <button type="button" className="c-vora__send" onClick={send} disabled={busy || finalizing || !input.trim()}>
          Send
        </button>
        {messages.length >= 2 && (
          <button type="button" className="c-vora__wrap" onClick={() => wrapUp(messages)} disabled={busy || finalizing}>
            Wrap it up
          </button>
        )}
      </div>
    </div>
  )
}
