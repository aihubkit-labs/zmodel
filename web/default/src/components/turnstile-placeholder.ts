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
export function getTurnstilePlaceholderCopy(props: {
  isReady: boolean
  isSlow: boolean
}) {
  if (props.isReady) {
    return null
  }

  if (props.isSlow) {
    return {
      title: 'Human verification is taking longer than expected',
      description:
        'If it does not appear for a long time, refresh the page and try again.',
    }
  }

  return {
    title: 'Human verification is loading',
    description: 'Please wait a moment...',
  }
}
