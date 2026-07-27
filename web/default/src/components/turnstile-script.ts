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
const TURNSTILE_SCRIPT_ID = 'cf-turnstile'
const TURNSTILE_SCRIPT_URL =
  'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit'

export function subscribeToTurnstileScript(
  onLoad: () => void,
  documentObject: Document = document
) {
  let script = documentObject.querySelector<HTMLScriptElement>(
    `#${TURNSTILE_SCRIPT_ID}`
  )

  if (!script) {
    script = documentObject.createElement('script')
    script.id = TURNSTILE_SCRIPT_ID
    script.src = TURNSTILE_SCRIPT_URL
    script.async = true
    script.defer = true
  }

  script.addEventListener('load', onLoad)

  if (!script.isConnected) {
    documentObject.head.appendChild(script)
  }

  return () => {
    script.removeEventListener('load', onLoad)
  }
}
