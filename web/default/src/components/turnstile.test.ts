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

import { getTurnstilePlaceholderCopy } from './turnstile-placeholder'
import { subscribeToTurnstileScript } from './turnstile-script'
import { subscribeToTurnstileWidget } from './turnstile-widget'

describe('getTurnstilePlaceholderCopy', () => {
  test('shows loading copy before the widget is rendered', () => {
    assert.deepEqual(
      getTurnstilePlaceholderCopy({
        isReady: false,
        isSlow: false,
      }),
      {
        title: 'Human verification is loading',
        description: 'Please wait a moment...',
      }
    )
  })

  test('shows refresh guidance when loading takes too long', () => {
    assert.deepEqual(
      getTurnstilePlaceholderCopy({
        isReady: false,
        isSlow: true,
      }),
      {
        title: 'Human verification is taking longer than expected',
        description:
          'If it does not appear for a long time, refresh the page and try again.',
      }
    )
  })

  test('hides the placeholder after the widget is rendered', () => {
    assert.equal(
      getTurnstilePlaceholderCopy({
        isReady: true,
        isSlow: false,
      }),
      null
    )
  })
})

describe('subscribeToTurnstileWidget', () => {
  test('reports a widget that is already rendered', () => {
    let readyCount = 0
    const container = {
      querySelector: () => ({ tagName: 'IFRAME' }),
    } as unknown as HTMLElement

    const unsubscribe = subscribeToTurnstileWidget(container, () => {
      readyCount += 1
    })

    assert.equal(readyCount, 1)
    unsubscribe()
  })

  test('reports the widget when its iframe is inserted', () => {
    let hasIframe = false
    let observerCallback: MutationCallback | undefined
    let disconnectCount = 0
    let readyCount = 0
    const container = {
      querySelector: () => (hasIframe ? { tagName: 'IFRAME' } : null),
    } as unknown as HTMLElement
    class MutationObserverStub {
      constructor(callback: MutationCallback) {
        observerCallback = callback
      }

      observe() {}

      disconnect() {
        disconnectCount += 1
      }
    }

    const unsubscribe = subscribeToTurnstileWidget(
      container,
      () => {
        readyCount += 1
      },
      MutationObserverStub
    )

    assert.equal(readyCount, 0)
    hasIframe = true
    observerCallback?.([], {} as MutationObserver)
    assert.equal(readyCount, 1)
    assert.equal(disconnectCount, 1)

    unsubscribe()
    assert.equal(disconnectCount, 1)
  })
})

describe('subscribeToTurnstileScript', () => {
  test('subscribes when the script already exists but is still loading', () => {
    const script = new EventTarget()
    Object.defineProperty(script, 'isConnected', { value: true })
    let loadCount = 0
    const documentStub = {
      querySelector: () => script,
    } as unknown as Document

    const unsubscribe = subscribeToTurnstileScript(() => {
      loadCount += 1
    }, documentStub)

    script.dispatchEvent(new Event('load'))
    assert.equal(loadCount, 1)

    unsubscribe()
    script.dispatchEvent(new Event('load'))
    assert.equal(loadCount, 1)
  })

  test('creates the script once and removes its listener on cleanup', () => {
    const script = new EventTarget() as EventTarget & {
      async: boolean
      defer: boolean
      id: string
      isConnected: boolean
      src: string
    }
    script.isConnected = false
    let appendedScript: typeof script | undefined
    let loadCount = 0
    const documentStub = {
      createElement: () => script,
      head: {
        appendChild: (element: typeof script) => {
          appendedScript = element
          element.isConnected = true
        },
      },
      querySelector: () => null,
    } as unknown as Document

    const unsubscribe = subscribeToTurnstileScript(() => {
      loadCount += 1
    }, documentStub)

    assert.equal(appendedScript, script)
    assert.equal(script.id, 'cf-turnstile')
    assert.equal(
      script.src,
      'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit'
    )

    unsubscribe()
    script.dispatchEvent(new Event('load'))
    assert.equal(loadCount, 0)
  })
})
