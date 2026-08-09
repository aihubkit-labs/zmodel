/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import type { TaskHTTPMessage } from '../types'

function formatHTTPBody(body?: string): string {
  if (!body) return '-'
  try {
    return JSON.stringify(JSON.parse(body), null, 2).replaceAll(
      /\r?\n/g,
      '\r\n'
    )
  } catch {
    return body.replaceAll(/\r?\n/g, '\r\n')
  }
}

function requestTarget(url?: string): string {
  if (!url) return '/'
  try {
    const parsed = new URL(url)
    return `${parsed.pathname}${parsed.search}` || '/'
  } catch {
    return url
  }
}

export function formatHTTPMessage(
  message: TaskHTTPMessage | undefined,
  transportErrorLabel: string
): string {
  if (!message) return '-'
  const protocol = message.protocol || 'HTTP/1.1'
  const headLines: string[] = []
  if (message.method) {
    headLines.push(
      `${message.method} ${requestTarget(message.url)} ${protocol}`
    )
  } else if (message.status || message.status_code) {
    const status = message.status || message.status_code
    headLines.push(`${protocol} ${status}`)
  }
  const headers = Object.entries(message.headers || {})
  if (headers.length > 0) {
    headers.sort(([left], [right]) => left.localeCompare(right))
    for (const [name, value] of headers) {
      headLines.push(`${name}: ${value}`)
    }
  }

  const sections: string[] = []
  if (headLines.length > 0) sections.push(headLines.join('\r\n'))
  if (message.body) sections.push(formatHTTPBody(message.body))
  if (message.error) {
    sections.push(`${transportErrorLabel}: ${message.error}`)
  }
  return sections.join('\r\n\r\n') || '-'
}
