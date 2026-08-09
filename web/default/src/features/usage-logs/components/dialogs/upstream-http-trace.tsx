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
import { Copy } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'

import { formatHTTPMessage } from '../../lib/http-trace'
import type { TaskHTTPMessage, TaskUpstreamHTTPTrace } from '../../types'

function HTTPMessageBlock(props: { label: string; message?: TaskHTTPMessage }) {
  const { t } = useTranslation()
  if (!props.message) return null

  const content = formatHTTPMessage(props.message, t('Transport error'))
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(content)
      toast.success(t('Copied'))
    } catch {
      toast.error(t('Copy failed'))
    }
  }

  return (
    <div className='min-w-0 space-y-2'>
      <div className='flex items-center justify-between gap-2'>
        <h5 className='text-muted-foreground text-xs font-medium'>
          {props.label}
        </h5>
        <Button
          type='button'
          variant='ghost'
          size='icon-xs'
          aria-label={t('Copy')}
          title={t('Copy')}
          onClick={copy}
        >
          <Copy />
        </Button>
      </div>
      <pre className='bg-muted/30 text-muted-foreground max-h-72 overflow-auto rounded-md border p-3 font-mono text-xs leading-relaxed break-words whitespace-pre-wrap'>
        {content}
      </pre>
      {props.message.body_truncated ? (
        <p className='text-muted-foreground text-xs'>
          {t('Content truncated at 64 KiB')}
        </p>
      ) : null}
    </div>
  )
}

function HTTPExchange(props: {
  title: string
  request?: TaskHTTPMessage
  response?: TaskHTTPMessage
}) {
  const { t } = useTranslation()
  if (!props.request && !props.response) return null

  return (
    <div className='space-y-3 border-t pt-3 first:border-t-0 first:pt-0'>
      <h4 className='text-sm font-medium'>{props.title}</h4>
      <HTTPMessageBlock label={t('Request')} message={props.request} />
      <HTTPMessageBlock label={t('Response')} message={props.response} />
    </div>
  )
}

export function UpstreamHTTPTrace(props: { trace: TaskUpstreamHTTPTrace }) {
  const { t } = useTranslation()

  return (
    <section className='space-y-3'>
      <h3 className='text-sm font-medium'>{t('Upstream HTTP diagnostics')}</h3>
      <div className='space-y-3'>
        <HTTPExchange
          title={t('Task submission')}
          request={props.trace.submit_request}
          response={props.trace.submit_response}
        />
        <HTTPExchange
          title={t('Failed task polling')}
          request={props.trace.poll_request}
          response={props.trace.poll_response}
        />
      </div>
    </section>
  )
}
