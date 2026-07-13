import React, { useEffect, useState } from "react"

interface ColorInputProps {
  value: string // #rrggbb
  onChange: (hex: string) => void
}

const HEX_RE = /^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/

// Native color picker + hex text field, two-way synced. The OS color dialog
// doesn't always expose a hex field (KDE shows RGB sliders) - the text input
// guarantees hex entry everywhere.
export const ColorInput = (props: ColorInputProps) => {
  const [text, setText] = useState(props.value)

  useEffect(() => {
    setText(props.value)
  }, [props.value])

  const handleText = (raw: string) => {
    let value = raw.trim()
    if (value !== "" && !value.startsWith("#")) {
      value = "#" + value
    }
    setText(value)
    if (HEX_RE.test(value)) {
      props.onChange(value.toLowerCase())
    }
  }

  const valid = HEX_RE.test(text)

  return (
    <span style={{ display: "inline-flex", alignItems: "center", gap: 8 }}>
      <input
        type="color"
        value={valid ? (text.length === 4 ? "#" + [...text.slice(1)].map((c) => c + c).join("") : text) : props.value}
        onChange={(e) => {
          setText(e.target.value)
          props.onChange(e.target.value)
        }}
        style={{ width: 34, height: 30, border: 0, background: "none", cursor: "pointer", padding: 0 }}
        aria-label="Pick a color"
      />
      <input
        type="text"
        value={text}
        onChange={(e) => handleText(e.target.value)}
        maxLength={7}
        spellCheck={false}
        aria-label="Hex color"
        style={{
          width: 92,
          font: "inherit",
          fontSize: 13,
          fontFamily: "monospace",
          padding: "5px 8px",
          borderRadius: 8,
          border: `1px solid ${valid ? "var(--colors-gray-300)" : "var(--colors-red-600)"}`,
          background: "transparent",
          color: "var(--colors-gray-900)",
          outline: "none",
        }}
        placeholder="#3B82F6"
      />
    </span>
  )
}
