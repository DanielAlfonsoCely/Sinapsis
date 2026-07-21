export function exportToCSV(rows: Record<string, string>[], filename: string) {
  if (rows.length === 0) return

  const headers = Object.keys(rows[0])
  const escape = (value: string) => {
    const needsQuotes = /[",\n]/.test(value)
    const escaped = value.replace(/"/g, '""')
    return needsQuotes ? `"${escaped}"` : escaped
  }

  const csvLines = [
    headers.join(","),
    ...rows.map((row) => headers.map((h) => escape(row[h] ?? "")).join(",")),
  ]

  const csvContent = "\uFEFF" + csvLines.join("\r\n") // BOM para acentos/ñ en Excel
  const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" })
  const url = URL.createObjectURL(blob)

  const link = document.createElement("a")
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

