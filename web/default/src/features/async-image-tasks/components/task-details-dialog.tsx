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
import { useQuery } from '@tanstack/react-query'
import dayjs from 'dayjs'
import { Download, ExternalLink, Image as ImageIcon } from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Dialog } from '@/components/dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { IconBadge } from '@/components/ui/icon-badge'
import { Skeleton } from '@/components/ui/skeleton'

import { getAsyncImageTaskDetail } from '../api'
import type {
  AsyncImageTask,
  AsyncImageTaskDetail,
  AsyncImageTaskObject,
} from '../types'

type AsyncImageTaskDetailsDialogProps = {
  task: AsyncImageTask | null
  root: boolean
  onClose: () => void
}

function formatTime(timestamp?: number) {
  return timestamp ? dayjs.unix(timestamp).format('YYYY-MM-DD HH:mm:ss') : '-'
}

function formatBytes(bytes: number) {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const unitIndex = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    units.length - 1
  )
  const value = bytes / 1024 ** unitIndex
  return `${value >= 10 || unitIndex === 0 ? value.toFixed(0) : value.toFixed(1)} ${units[unitIndex]}`
}

function DetailItem(props: { label: string; children: ReactNode }) {
  return (
    <div className='min-w-0 border-b py-2.5 last:border-b-0 sm:grid sm:grid-cols-[10rem_minmax(0,1fr)] sm:gap-4'>
      <dt className='text-muted-foreground mb-1 text-xs sm:mb-0'>
        {props.label}
      </dt>
      <dd className='min-w-0 text-sm break-words'>{props.children}</dd>
    </div>
  )
}

function StatusValue(props: { value: string }) {
  const { t } = useTranslation()
  return <Badge variant='outline'>{t(props.value || 'unknown')}</Badge>
}

function ObjectPreview(props: {
  object: AsyncImageTaskObject
  taskId: string
  root: boolean
}) {
  const { t } = useTranslation()
  const imageLabel = t('Image {{index}}', { index: props.object.index + 1 })

  return (
    <article className='min-w-0 overflow-hidden rounded-md border'>
      <div className='bg-muted/30 flex aspect-[4/3] items-center justify-center overflow-hidden'>
        {props.object.preview_url ? (
          <a
            href={props.object.preview_url}
            target='_blank'
            rel='noreferrer'
            className='flex size-full items-center justify-center'
            aria-label={`${t('Open in new tab')}: ${imageLabel}`}
          >
            <img
              src={props.object.preview_url}
              alt={imageLabel}
              loading='lazy'
              className='size-full object-contain'
            />
          </a>
        ) : (
          <div className='text-muted-foreground flex flex-col items-center gap-2 px-4 text-center text-sm'>
            <ImageIcon className='size-8' />
            <span>{t('Preview unavailable')}</span>
          </div>
        )}
      </div>
      <div className='space-y-3 p-3'>
        <div className='flex flex-wrap items-center justify-between gap-2'>
          <span className='font-medium'>{imageLabel}</span>
          <span className='text-muted-foreground text-xs'>
            {formatBytes(props.object.size_bytes)}
          </span>
        </div>
        <div className='flex flex-wrap gap-1.5'>
          <StatusValue value={props.object.status} />
          <StatusValue value={props.object.staging_status} />
          {props.object.mime_type ? (
            <Badge variant='secondary'>{props.object.mime_type}</Badge>
          ) : null}
        </div>
        <div className='flex flex-wrap gap-2'>
          <Button
            variant='outline'
            size='sm'
            disabled={!props.object.preview_url}
            render={
              props.object.preview_url ? (
                <a
                  href={props.object.preview_url}
                  target='_blank'
                  rel='noreferrer'
                />
              ) : undefined
            }
          >
            <ExternalLink />
            {t('Preview')}
          </Button>
          <Button
            size='sm'
            disabled={!props.object.download_url}
            render={
              props.object.download_url ? (
                <a
                  href={props.object.download_url}
                  download={`${props.taskId}-${props.object.index}.${props.object.extension}`}
                />
              ) : undefined
            }
          >
            <Download />
            {t('Download image')}
          </Button>
        </div>
        <dl className='border-t pt-1'>
          <DetailItem label={t('Storage status')}>
            {t(props.object.status)}
          </DetailItem>
          <DetailItem label={t('Staging status')}>
            {t(props.object.staging_status)}
          </DetailItem>
          <DetailItem label={t('File size')}>
            {formatBytes(props.object.size_bytes)}
          </DetailItem>
          <DetailItem label={t('MIME type')}>
            {props.object.mime_type || '-'}
          </DetailItem>
          <DetailItem label={t('File extension')}>
            {props.object.extension || '-'}
          </DetailItem>
          <DetailItem label={t('Staging file size')}>
            {formatBytes(props.object.staging_size_bytes)}
          </DetailItem>
          <DetailItem label={t('Staged at')}>
            {formatTime(props.object.staged_at)}
          </DetailItem>
          <DetailItem label={t('Uploaded at')}>
            {formatTime(props.object.uploaded_at)}
          </DetailItem>
          <DetailItem label={t('Expires at')}>
            {formatTime(props.object.expires_at)}
          </DetailItem>
          <DetailItem label={t('Deleted at')}>
            {formatTime(props.object.deleted_at)}
          </DetailItem>
          <DetailItem label={t('Delete attempts')}>
            {props.object.delete_attempts}
          </DetailItem>
          {props.root && props.object.provider ? (
            <DetailItem label={t('Provider')}>
              {props.object.provider}
            </DetailItem>
          ) : null}
          {props.root && props.object.endpoint ? (
            <DetailItem label={t('Endpoint')}>
              <span className='font-mono text-xs'>{props.object.endpoint}</span>
            </DetailItem>
          ) : null}
          {props.root && props.object.region ? (
            <DetailItem label={t('Region')}>{props.object.region}</DetailItem>
          ) : null}
          {props.root && props.object.object_key ? (
            <DetailItem label={t('Object key')}>
              <span className='font-mono text-xs'>
                {props.object.object_key}
              </span>
            </DetailItem>
          ) : null}
          {props.root && props.object.bucket ? (
            <DetailItem label='Bucket'>{props.object.bucket}</DetailItem>
          ) : null}
          {props.root && props.object.etag ? (
            <DetailItem label='ETag'>{props.object.etag}</DetailItem>
          ) : null}
          {props.root && props.object.last_error ? (
            <DetailItem label={t('Error')}>
              <span className='text-destructive'>
                {props.object.last_error}
              </span>
            </DetailItem>
          ) : null}
        </dl>
      </div>
    </article>
  )
}

function LoadedTaskDetail(props: {
  detail: AsyncImageTaskDetail
  root: boolean
}) {
  const { t } = useTranslation()
  const detail = props.detail

  return (
    <div className='space-y-5'>
      <TaskImages detail={detail} root={props.root} />

      <div className='grid gap-5 lg:grid-cols-2'>
        <section className='space-y-2'>
          <h3 className='text-sm font-medium'>{t('Generation details')}</h3>
          <dl className='rounded-md border px-3'>
            <DetailItem label={t('Task ID')}>
              <span className='inline-flex max-w-full items-center gap-1'>
                <span className='truncate font-mono text-xs'>
                  {detail.task_id}
                </span>
                <CopyButton value={detail.task_id} className='size-7' />
              </span>
            </DetailItem>
            {props.root ? (
              <DetailItem label={t('User')}>
                {detail.username || detail.user_id}
              </DetailItem>
            ) : null}
            {props.root ? (
              <DetailItem label={t('User ID')}>{detail.user_id}</DetailItem>
            ) : null}
            <DetailItem label={t('Model')}>{detail.model}</DetailItem>
            <DetailItem label={t('Group')}>
              {detail.using_group || '-'}
            </DetailItem>
            {props.root ? (
              <DetailItem label={t('Channel')}>
                {detail.channel_name || '-'}
              </DetailItem>
            ) : null}
            {props.root ? (
              <DetailItem label={t('Channel ID')}>
                {detail.last_channel_id || '-'}
              </DetailItem>
            ) : null}
            {props.root ? (
              <DetailItem label={t('Platform')}>
                {detail.platform || '-'}
              </DetailItem>
            ) : null}
            {props.root ? (
              <DetailItem label={t('Token ID')}>
                {detail.token_id || '-'}
              </DetailItem>
            ) : null}
            {props.root ? (
              <DetailItem label={t('Subscription ID')}>
                {detail.subscription_id || '-'}
              </DetailItem>
            ) : null}
            <DetailItem label={t('Generation status')}>
              <StatusValue value={detail.status} />
            </DetailItem>
            <DetailItem label={t('Upload status')}>
              <StatusValue
                value={
                  detail.output_availability === 'available'
                    ? 'Uploaded'
                    : detail.output_availability
                }
              />
            </DetailItem>
            <DetailItem label={t('Source kind')}>
              {t(detail.source_kind || 'none')}
            </DetailItem>
            {detail.error ? (
              <DetailItem label={t('Error')}>
                <span className='text-destructive'>{detail.error}</span>
              </DetailItem>
            ) : null}
            {detail.error_code ? (
              <DetailItem label={t('Error code')}>
                <span className='font-mono text-xs'>{detail.error_code}</span>
              </DetailItem>
            ) : null}
          </dl>
        </section>

        <section className='space-y-2'>
          <h3 className='text-sm font-medium'>{t('Archive processing')}</h3>
          <dl className='rounded-md border px-3'>
            <DetailItem label={t('Object retention (seconds)')}>
              {detail.retention_seconds}
            </DetailItem>
            <DetailItem label={t('Archive attempt timeout (seconds)')}>
              {detail.archive_timeout_seconds}
            </DetailItem>
            <DetailItem label={t('Maximum archive attempts')}>
              {detail.archive_max_attempts}
            </DetailItem>
            <DetailItem label={t('Archive attempts')}>
              {detail.archive_attempts}
            </DetailItem>
          </dl>
        </section>

        <section className='space-y-2'>
          <h3 className='text-sm font-medium'>{t('Billing')}</h3>
          <dl className='rounded-md border px-3'>
            <DetailItem label={t('Billing status')}>
              <StatusValue value={detail.billing_status} />
            </DetailItem>
            <DetailItem label={t('Billing source')}>
              {t(detail.billing_source || 'none')}
            </DetailItem>
            <DetailItem label={t('Reserved quota')}>
              {detail.reserved_quota}
            </DetailItem>
            <DetailItem label={t('Actual quota')}>
              {detail.actual_quota}
            </DetailItem>
          </dl>
        </section>

        <section className='space-y-2'>
          <h3 className='text-sm font-medium'>{t('Lifecycle')}</h3>
          <dl className='rounded-md border px-3'>
            <DetailItem label={t('Created at')}>
              {formatTime(detail.created_at)}
            </DetailItem>
            <DetailItem label={t('Started at')}>
              {formatTime(detail.started_at)}
            </DetailItem>
            <DetailItem label={t('Generation completed at')}>
              {formatTime(detail.generation_completed_at)}
            </DetailItem>
            <DetailItem label={t('Billing finalized at')}>
              {formatTime(detail.billing_finalized_at)}
            </DetailItem>
            <DetailItem label={t('Completed at')}>
              {formatTime(detail.completed_at)}
            </DetailItem>
            <DetailItem label={t('Updated at')}>
              {formatTime(detail.updated_at)}
            </DetailItem>
            <DetailItem label={t('Archive retry deadline')}>
              {formatTime(detail.archive_retry_deadline_at)}
            </DetailItem>
            <DetailItem label={t('Next retry')}>
              {formatTime(detail.next_attempt_at)}
            </DetailItem>
            <DetailItem label={t('Output expires at')}>
              {formatTime(detail.output_expires_at)}
            </DetailItem>
            <DetailItem label={t('Manually recovered at')}>
              {formatTime(detail.manually_recovered_at)}
            </DetailItem>
          </dl>
        </section>

        <section className='space-y-2 lg:col-span-2'>
          <h3 className='text-sm font-medium'>{t('Request parameters')}</h3>
          <pre className='bg-muted/30 text-muted-foreground max-h-80 overflow-auto rounded-md border p-3 font-mono text-xs leading-relaxed break-words whitespace-pre-wrap'>
            {detail.request ? JSON.stringify(detail.request, null, 2) : '-'}
          </pre>
        </section>
      </div>
    </div>
  )
}

function TaskImages(props: { detail: AsyncImageTaskDetail; root: boolean }) {
  const { t } = useTranslation()

  return (
    <section className='space-y-3'>
      <h3 className='text-sm font-medium'>{t('Images')}</h3>
      {props.detail.objects.length > 0 ? (
        <div className='grid gap-3 md:grid-cols-2'>
          {props.detail.objects.map((object) => (
            <ObjectPreview
              key={object.index}
              object={object}
              taskId={props.detail.task_id}
              root={props.root}
            />
          ))}
        </div>
      ) : (
        <div className='text-muted-foreground rounded-md border py-12 text-center text-sm'>
          {t('No image is available for preview or download')}
        </div>
      )}
    </section>
  )
}

export function AsyncImageTaskDetailsDialog(
  props: AsyncImageTaskDetailsDialogProps
) {
  const { t } = useTranslation()
  const taskId = props.task?.task_id ?? ''
  const detailsQuery = useQuery({
    queryKey: ['async-image-task-detail', props.root, taskId],
    queryFn: () => getAsyncImageTaskDetail(taskId, props.root),
    enabled: taskId !== '',
  })
  const detail = detailsQuery.data

  let content: ReactNode
  if (detailsQuery.isLoading) {
    content = (
      <div className='space-y-3'>
        <Skeleton className='h-48 w-full' />
        <Skeleton className='h-36 w-full' />
      </div>
    )
  } else if (detailsQuery.isError || !detail) {
    content = (
      <div className='text-muted-foreground py-16 text-center text-sm'>
        {t('Failed to load')}
      </div>
    )
  } else {
    content = <LoadedTaskDetail detail={detail} root={props.root} />
  }

  return (
    <Dialog
      open={props.task !== null}
      onOpenChange={(open) => {
        if (!open) props.onClose()
      }}
      title={
        <>
          <IconBadge tone='primary' size='sm'>
            <ImageIcon />
          </IconBadge>
          {t('Task details')}
        </>
      }
      description={taskId ? `${t('Task ID:')} ${taskId}` : undefined}
      titleClassName='flex items-center gap-2'
      contentClassName='w-[min(96vw,72rem)] sm:max-w-6xl'
      contentHeight='min(78vh,58rem)'
    >
      {content}
    </Dialog>
  )
}
