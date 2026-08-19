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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { ClipboardList, ExternalLink, Eye, Upload } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { IconBadge } from '@/components/ui/icon-badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Spinner } from '@/components/ui/spinner'
import { formatLogQuota, formatTimestampToDate } from '@/lib/format'

import { retryVideoUpload } from '../../api'
import { taskActionMapper, taskStatusMapper } from '../../lib/mappers'
import { canUploadVideoTaskToS3, isVideoTask } from '../../lib/video-task'
import type { TaskLog } from '../../types'
import { UpstreamHTTPTrace } from './upstream-http-trace'

interface TaskDetailsDialogProps {
  log: TaskLog
  isAdmin: boolean
  open: boolean
  onOpenChange: (open: boolean) => void
}

interface TaskRequestSnapshot {
  images?: string[]
  reference_images?: string[]
  reference_videos?: string[]
  reference_audios?: string[]
  first_image?: string
  last_image?: string
}

type MediaType = 'image' | 'video' | 'audio'

interface MediaItem {
  type: MediaType
  url: string
}

function DetailItem(props: { label: string; children: ReactNode }) {
  return (
    <div className='min-w-0 border-b py-2.5 last:border-b-0 sm:grid sm:grid-cols-[8rem_minmax(0,1fr)] sm:gap-4'>
      <dt className='text-muted-foreground mb-1 text-xs sm:mb-0'>
        {props.label}
      </dt>
      <dd className='min-w-0 text-sm break-words'>{props.children}</dd>
    </div>
  )
}

function formatTaskTime(timestamp?: number) {
  return timestamp ? formatTimestampToDate(timestamp, 'seconds') : '-'
}

function parseJSONValue(value: unknown): unknown {
  if (typeof value !== 'string') return value
  if (!value.trim()) return null
  try {
    return JSON.parse(value)
  } catch {
    return value
  }
}

function getMediaItems(snapshot: unknown): MediaItem[] {
  if (!snapshot || typeof snapshot !== 'object' || Array.isArray(snapshot)) {
    return []
  }

  const request = snapshot as TaskRequestSnapshot
  const items: MediaItem[] = []
  const seen = new Set<string>()
  const append = (type: MediaType, urls?: string[]) => {
    if (!Array.isArray(urls)) return
    for (const url of urls) {
      const key = `${type}:${url}`
      if (typeof url === 'string' && url && !seen.has(key)) {
        seen.add(key)
        items.push({ type, url })
      }
    }
  }

  append('image', request.images)
  append('image', request.reference_images)
  append(
    'image',
    [request.first_image, request.last_image].filter(Boolean) as string[]
  )
  append('video', request.reference_videos)
  append('audio', request.reference_audios)
  return items
}

function MediaPreview(props: { item: MediaItem; index: number }) {
  const { t } = useTranslation()
  const [loaded, setLoaded] = useState(false)
  let label = t('Audio')
  if (props.item.type === 'image') {
    label = t('Image')
  } else if (props.item.type === 'video') {
    label = t('Video')
  }

  return (
    <div className='min-w-0 space-y-2 rounded-md border p-3'>
      <div className='flex min-w-0 items-center justify-between gap-2'>
        <span className='text-xs font-medium'>
          {label} {props.index + 1}
        </span>
        <div className='flex shrink-0 items-center gap-1'>
          {!loaded ? (
            <Button
              type='button'
              variant='outline'
              size='xs'
              onClick={() => setLoaded(true)}
            >
              <Eye />
              {t('Preview')}
            </Button>
          ) : null}
          <Button
            variant='ghost'
            size='icon-xs'
            aria-label={t('Open in new tab')}
            title={t('Open in new tab')}
            render={
              <a href={props.item.url} target='_blank' rel='noreferrer' />
            }
          >
            <ExternalLink />
          </Button>
        </div>
      </div>
      <a
        href={props.item.url}
        target='_blank'
        rel='noreferrer'
        className='text-muted-foreground block truncate font-mono text-[11px] hover:underline'
        title={props.item.url}
      >
        {props.item.url}
      </a>
      {loaded && props.item.type === 'image' ? (
        <img
          src={props.item.url}
          alt={label}
          loading='lazy'
          className='max-h-64 w-full rounded-md bg-black object-contain'
        />
      ) : null}
      {loaded && props.item.type === 'video' ? (
        <video
          src={props.item.url}
          controls
          playsInline
          preload='metadata'
          className='max-h-64 w-full rounded-md bg-black object-contain'
        />
      ) : null}
      {loaded && props.item.type === 'audio' ? (
        <audio
          src={props.item.url}
          controls
          preload='metadata'
          className='w-full'
        />
      ) : null}
    </div>
  )
}

export function TaskDetailsDialog(props: TaskDetailsDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const uploadMutation = useMutation({
    mutationFn: () => retryVideoUpload(props.log.task_id),
    onSuccess: () => {
      toast.success(t('Video upload started'))
      queryClient.invalidateQueries({ queryKey: ['logs'] })
    },
    onError: () => {
      toast.error(t('Failed to upload video to S3'))
    },
  })
  const model =
    props.log.properties?.origin_model_name ||
    (props.isAdmin ? props.log.properties?.upstream_model_name : undefined) ||
    '-'
  const request = parseJSONValue(props.log.properties?.input)
  const mediaItems = getMediaItems(request)
  const tokenUsage = props.log.token_usage
  const canUploadVideo = props.isAdmin && canUploadVideoTaskToS3(props.log)
  const showVideoStorage =
    props.isAdmin &&
    isVideoTask(props.log) &&
    Boolean(props.log.video_storage_status || canUploadVideo)
  const videoStorageStatusLabels: Record<string, string> = {
    available: t('Available'),
    uploading: t('Uploading'),
    failed: t('Failed'),
    expired: t('Expired'),
    deleted: t('Deleted'),
  }
  let videoStorageVariant: 'success' | 'info' | 'danger' | 'neutral' = 'neutral'
  if (props.log.video_storage_status === 'available') {
    videoStorageVariant = 'success'
  } else if (props.log.video_storage_status === 'uploading') {
    videoStorageVariant = 'info'
  } else if (props.log.video_storage_status) {
    videoStorageVariant = 'danger'
  }

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={
        <>
          <IconBadge tone='info' size='sm'>
            <ClipboardList />
          </IconBadge>
          {t('Details')}
        </>
      }
      description={`${t('Task ID:')} ${props.log.task_id}`}
      titleClassName='flex items-center gap-2'
      contentClassName='max-h-[90vh] overflow-hidden sm:max-w-3xl'
      contentHeight='auto'
    >
      <ScrollArea className='max-h-[72vh] pr-3'>
        <div className='space-y-4'>
          <dl className='rounded-md border px-3'>
            <DetailItem label={t('Task ID')}>
              <StatusBadge
                label={props.log.task_id}
                copyText={props.log.task_id}
                variant='neutral'
                size='sm'
                className='max-w-full font-mono'
              />
            </DetailItem>
            <DetailItem label={t('Status')}>
              <StatusBadge
                label={t(
                  taskStatusMapper.getLabel(
                    props.log.status,
                    props.log.status || 'Submitting'
                  )
                )}
                variant={taskStatusMapper.getVariant(props.log.status)}
                size='sm'
                copyable={false}
              />
            </DetailItem>
            <DetailItem label={t('Group')}>{props.log.group || '-'}</DetailItem>
            <DetailItem label={t('Platform')}>
              {t(props.log.platform_name || props.log.platform)}
            </DetailItem>
            <DetailItem label={t('Type')}>
              {t(taskActionMapper.getLabel(props.log.action))}
            </DetailItem>
            <DetailItem label={t('Model')}>
              <span className='font-mono text-xs'>{model}</span>
            </DetailItem>
            <DetailItem label={t('Progress')}>
              <span className='font-mono text-xs'>
                {props.log.progress || '-'}
              </span>
            </DetailItem>
            {tokenUsage ? (
              <>
                <DetailItem label={t('Total Tokens')}>
                  <span className='font-mono text-xs tabular-nums'>
                    {tokenUsage.total_tokens?.toLocaleString() || '-'}
                  </span>
                </DetailItem>
                {tokenUsage.input_tokens != null ? (
                  <DetailItem label={t('Input Tokens')}>
                    <span className='font-mono text-xs tabular-nums'>
                      {tokenUsage.input_tokens.toLocaleString()}
                    </span>
                  </DetailItem>
                ) : null}
                {tokenUsage.output_tokens != null ? (
                  <DetailItem label={t('Output Tokens')}>
                    <span className='font-mono text-xs tabular-nums'>
                      {tokenUsage.output_tokens.toLocaleString()}
                    </span>
                  </DetailItem>
                ) : null}
              </>
            ) : null}
            <DetailItem label={t('Total Cost')}>
              <span className='font-mono text-xs tabular-nums'>
                {formatLogQuota(props.log.quota)}
              </span>
            </DetailItem>
            <DetailItem label={t('Submit Time')}>
              <span className='font-mono text-xs tabular-nums'>
                {formatTaskTime(props.log.submit_time)}
              </span>
            </DetailItem>
            <DetailItem label={t('Start Time')}>
              <span className='font-mono text-xs tabular-nums'>
                {formatTaskTime(props.log.start_time)}
              </span>
            </DetailItem>
            <DetailItem label={t('End Time')}>
              <span className='font-mono text-xs tabular-nums'>
                {formatTaskTime(props.log.finish_time)}
              </span>
            </DetailItem>
            {props.log.username ? (
              <DetailItem label={t('User')}>{props.log.username}</DetailItem>
            ) : null}
            {props.log.channel_name ? (
              <DetailItem label={t('Channel')}>
                {props.log.channel_name}
              </DetailItem>
            ) : null}
            {props.log.fail_reason ? (
              <DetailItem label={t('Fail Reason')}>
                <span className='text-destructive whitespace-pre-wrap'>
                  {props.log.fail_reason}
                </span>
              </DetailItem>
            ) : null}
            {showVideoStorage ? (
              <DetailItem label={t('Upload status')}>
                <div className='flex flex-wrap items-center gap-2'>
                  <StatusBadge
                    label={
                      videoStorageStatusLabels[
                        props.log.video_storage_status || ''
                      ] || t('Not uploaded')
                    }
                    variant={videoStorageVariant}
                    size='sm'
                    copyable={false}
                  />
                  {canUploadVideo ? (
                    <Button
                      type='button'
                      variant='outline'
                      size='xs'
                      disabled={uploadMutation.isPending}
                      onClick={() => uploadMutation.mutate()}
                    >
                      {uploadMutation.isPending ? (
                        <Spinner data-icon='inline-start' />
                      ) : (
                        <Upload data-icon='inline-start' />
                      )}
                      {t('Upload to S3')}
                    </Button>
                  ) : null}
                </div>
              </DetailItem>
            ) : null}
            {props.isAdmin && props.log.video_storage_error ? (
              <DetailItem label={t('Storage error')}>
                <span className='text-destructive whitespace-pre-wrap'>
                  {props.log.video_storage_error}
                </span>
              </DetailItem>
            ) : null}
          </dl>

          {mediaItems.length > 0 ? (
            <section className='space-y-2'>
              <h3 className='text-sm font-medium'>{t('Input')}</h3>
              <div className='grid gap-2 sm:grid-cols-2'>
                {mediaItems.map((item, index) => (
                  <MediaPreview
                    key={`${item.type}-${item.url}`}
                    item={item}
                    index={index}
                  />
                ))}
              </div>
            </section>
          ) : null}

          {props.isAdmin ? (
            <UpstreamHTTPTrace trace={props.log.upstream_http_trace} />
          ) : null}
        </div>
      </ScrollArea>
    </Dialog>
  )
}
