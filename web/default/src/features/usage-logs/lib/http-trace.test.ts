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

import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { formatHTTPMessage } from './http-trace'

describe('upstream HTTP trace formatting', () => {
  test('reconstructs a normalized HTTP request with CRLF separators', () => {
    const message = formatHTTPMessage(
      {
        method: 'POST',
        url: 'https://upstream.example/v1/videos?mode=create',
        protocol: 'HTTP/1.1',
        headers: {
          Authorization: '[REDACTED]',
          'Content-Type': 'application/json',
          Host: 'upstream.example',
        },
        body: '{"model":"seedance-2.5"}',
      },
      'Transport error'
    )

    assert.equal(
      message,
      [
        'POST /v1/videos?mode=create HTTP/1.1',
        'Authorization: [REDACTED]',
        'Content-Type: application/json',
        'Host: upstream.example',
        '',
        '{',
        '  "model": "seedance-2.5"',
        '}',
      ].join('\r\n')
    )
  })

  test('reconstructs a normalized HTTP response status line', () => {
    const message = formatHTTPMessage(
      {
        protocol: 'HTTP/2.0',
        status: '403 Forbidden',
        status_code: 403,
        headers: { 'Content-Type': 'application/json' },
        body: '{"error":"forbidden"}',
      },
      'Transport error'
    )

    assert.equal(
      message,
      [
        'HTTP/2.0 403 Forbidden',
        'Content-Type: application/json',
        '',
        '{',
        '  "error": "forbidden"',
        '}',
      ].join('\r\n')
    )
  })
})
