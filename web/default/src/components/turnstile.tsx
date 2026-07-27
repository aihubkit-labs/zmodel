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
import { useEffect, useEffectEvent, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Spinner } from '@/components/ui/spinner'
import { cn } from '@/lib/utils'

import { getTurnstilePlaceholderCopy } from './turnstile-placeholder'
import { subscribeToTurnstileScript } from './turnstile-script'
import { subscribeToTurnstileWidget } from './turnstile-widget'

declare global {
  interface Window {
    turnstile?: {
      render: (
        element: HTMLElement,
        options: Record<string, unknown>
      ) => string | undefined
      remove?: (widgetId: string) => void
    }
  }
}

const TURNSTILE_SLOW_LOAD_MS = 8000

interface TurnstileProps {
  siteKey: string
  onVerify: (token: string) => void
  onExpire?: () => void
  className?: string
}

export function Turnstile(props: TurnstileProps) {
  const { t } = useTranslation()
  const ref = useRef<HTMLDivElement | null>(null)
  const [isReady, setIsReady] = useState(false)
  const [isSlow, setIsSlow] = useState(false)
  const onExpire = useEffectEvent(() => props.onExpire?.())
  const onVerify = useEffectEvent(props.onVerify)

  const placeholderCopy = getTurnstilePlaceholderCopy({ isReady, isSlow })

  useEffect(() => {
    let isActive = true
    let widgetId: string | undefined
    let unsubscribeFromScript = () => {}
    let unsubscribeFromWidget = () => {}

    setIsReady(false)
    setIsSlow(false)

    const slowTimer = window.setTimeout(() => {
      if (isActive) {
        setIsSlow(true)
      }
    }, TURNSTILE_SLOW_LOAD_MS)

    const render = () => {
      if (!isActive || !ref.current || !window.turnstile) return

      unsubscribeFromWidget = subscribeToTurnstileWidget(ref.current, () => {
        if (!isActive) return

        window.clearTimeout(slowTimer)
        setIsSlow(false)
        setIsReady(true)
      })

      try {
        widgetId = window.turnstile.render(ref.current, {
          sitekey: props.siteKey,
          callback: (token: string) => {
            if (!isActive) return

            onVerify(token)
          },
          'error-callback': () => {
            if (isActive) {
              onExpire()
            }
          },
          'expired-callback': () => {
            if (isActive) {
              onExpire()
            }
          },
        })
      } catch {
        unsubscribeFromWidget()
      }
    }

    if (window.turnstile) {
      render()
    } else {
      unsubscribeFromScript = subscribeToTurnstileScript(render)
    }

    return () => {
      isActive = false
      window.clearTimeout(slowTimer)
      unsubscribeFromScript()
      unsubscribeFromWidget()

      if (widgetId) {
        window.turnstile?.remove?.(widgetId)
      }
    }
  }, [props.siteKey])

  return (
    <div
      className={cn(
        'relative transition-[min-height] duration-200',
        isReady ? 'min-h-0' : 'min-h-[66px]',
        props.className
      )}
    >
      {placeholderCopy ? (
        <div
          className='bg-background text-muted-foreground pointer-events-none absolute inset-0 z-0 flex min-h-[66px] items-center gap-3 rounded-md px-4 py-3'
          role='status'
        >
          <Spinner className='size-4 shrink-0' aria-hidden='true' />
          <div className='min-w-0 space-y-0.5'>
            <p className='text-foreground text-sm font-medium'>
              {t(placeholderCopy.title)}
            </p>
            <p className='text-xs leading-5'>
              {t(placeholderCopy.description)}
            </p>
          </div>
        </div>
      ) : null}

      <div
        ref={ref}
        className={cn(
          'relative z-10 transition-[min-height] duration-200',
          isReady ? 'min-h-0' : 'min-h-[66px]'
        )}
      />
    </div>
  )
}
