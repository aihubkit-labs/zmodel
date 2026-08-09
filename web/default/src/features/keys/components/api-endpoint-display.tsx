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
import { Copy01Icon, Tick02Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import type { SystemStatus } from '@/features/auth/types'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { useStatus } from '@/hooks/use-status'

function resolveApiEndpoint(status: SystemStatus | null): string {
  const candidates = [
    status?.api_server_address,
    status?.data?.api_server_address,
    status?.server_address,
    status?.data?.server_address,
  ]
  const configuredEndpoint = candidates.find(
    (value): value is string =>
      typeof value === 'string' && value.trim().length > 0
  )
  const fallbackEndpoint =
    typeof window === 'undefined' ? '' : window.location.origin

  return (configuredEndpoint ?? fallbackEndpoint).trim().replace(/\/+$/, '')
}

export function ApiEndpointDisplay() {
  const { t } = useTranslation()
  const { status } = useStatus()
  const { copiedText, copyToClipboard } = useCopyToClipboard()
  const endpoint = resolveApiEndpoint(status)
  const isCopied = copiedText === endpoint

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant='outline'
            className='h-9 w-full min-w-0 justify-start px-3 sm:w-fit sm:max-w-full'
            onClick={() => copyToClipboard(endpoint)}
            aria-label={`${t('Copy to clipboard')}: ${endpoint}`}
            disabled={!endpoint}
          />
        }
      >
        <span className='text-muted-foreground shrink-0'>
          {t('API Endpoint')}
        </span>
        <Separator orientation='vertical' className='h-4!' />
        <code className='min-w-0 flex-1 truncate text-left text-xs sm:max-w-96 sm:flex-none sm:text-sm'>
          {endpoint}
        </code>
        <HugeiconsIcon
          icon={isCopied ? Tick02Icon : Copy01Icon}
          data-icon='inline-end'
          strokeWidth={2}
          className={isCopied ? 'text-success' : undefined}
        />
      </TooltipTrigger>
      <TooltipContent>
        {isCopied ? t('Copied!') : t('Copy to clipboard')}
      </TooltipContent>
    </Tooltip>
  )
}
