import "./VoraChat.scss"

import React, { useEffect, useRef, useState } from "react"
import { AIMessage } from "@fider/models"
import { actions, Fider, notify } from "@fider/services"

interface VoraChatProps {
  productId: number
  onDone: (title: string, description: string, brief: string) => void
  onClose: () => void
}

// Vora — the ideation agent. Pure conversation: the only outputs are text
// and, at the end, a {title, description, brief} the user reviews.
export const VoraChat = (props: VoraChatProps) => {
  const agent = Fider.session.tenant.aiAgents?.find((a) => a.productId === props.productId) ?? Fider.session.tenant.aiAgents?.find((a) => a.productId === null)

  const [messages, setMessages] = useState<AIMessage[]>([])
  const [input, setInput] = useState("")
  const [busy, setBusy] = useState(false)
  const [finalizing, setFinalizing] = useState(false)
  const logRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    logRef.current?.scrollTo({ top: logRef.current.scrollHeight })
  }, [messages, busy])

  const send = async () => {
    const text = input.trim()
    if (!text || busy) {
      return
    }
    const next: AIMessage[] = [...messages, { role: "user", content: text }]
    setMessages(next)
    setInput("")
    setBusy(true)
    const result = await actions.aiIdeate(props.productId, next)
    setBusy(false)
    if (result.ok) {
      setMessages([...next, { role: "assistant", content: result.data.reply }])
    } else {
      notify.error(result.error?.errors?.[0]?.message || "Vora didn't answer — try again.")
      setMessages(messages)
      setInput(text)
    }
  }

  const wrapUp = async () => {
    if (busy || finalizing || messages.length < 2) {
      return
    }
    setFinalizing(true)
    const result = await actions.aiFinalize(props.productId, messages)
    setFinalizing(false)
    if (result.ok) {
      props.onDone(result.data.title, result.data.description, result.data.brief)
    } else {
      notify.error(result.error?.errors?.[0]?.message || "Vora couldn't wrap up — try once more.")
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
            {m.content}
          </div>
        ))}
        {(busy || finalizing) && <div className="c-vora__thinking">{finalizing ? "Vora is drafting your idea…" : "Vora is thinking…"}</div>}
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
          <button type="button" className="c-vora__wrap" onClick={wrapUp} disabled={busy || finalizing}>
            Wrap it up
          </button>
        )}
      </div>
    </div>
  )
}
