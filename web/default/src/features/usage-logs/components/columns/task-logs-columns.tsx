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
import type { ColumnDef } from '@tanstack/react-table'
import { Eye, Info, Music } from 'lucide-react'
/* eslint-disable react-refresh/only-export-components */
import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { getUserAvatarFallback, getUserAvatarStyle } from '@/lib/avatar'
import { formatTimestampToDate } from '@/lib/format'
import { cn } from '@/lib/utils'

import { TASK_ACTIONS, TASK_STATUS } from '../../constants'
import { taskActionMapper, taskStatusMapper } from '../../lib/mappers'
import type { TaskLog } from '../../types'
import {
  AudioPreviewDialog,
  type AudioClip,
} from '../dialogs/audio-preview-dialog'
import { FailReasonDialog } from '../dialogs/fail-reason-dialog'
import { TaskDetailsDialog } from '../dialogs/task-details-dialog'
import { VideoPreviewDialog } from '../dialogs/video-preview-dialog'
import { useUsageLogsContext } from '../usage-logs-provider'
import {
  createDurationColumn,
  createChannelColumn,
  createProgressColumn,
} from './column-helpers'

function parseTaskData(data: unknown): unknown[] {
  if (Array.isArray(data)) return data
  if (typeof data === 'string') {
    try {
      const parsed = JSON.parse(data)
      return Array.isArray(parsed) ? parsed : []
    } catch {
      return []
    }
  }
  return []
}

function AudioPreviewCell({ log }: { log: TaskLog }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const clips = useMemo(() => {
    const data = parseTaskData(log.data)
    return data.filter(
      (c) =>
        c && typeof c === 'object' && (c as Record<string, unknown>).audio_url
    )
  }, [log.data])

  if (clips.length === 0) return null

  return (
    <>
      <button
        type='button'
        className='group flex items-center gap-1 text-left text-xs'
        onClick={() => setOpen(true)}
      >
        <Music className='text-muted-foreground size-3' />
        <span className='text-foreground leading-snug group-hover:underline'>
          {t('Click to preview audio')}
        </span>
      </button>
      <AudioPreviewDialog
        open={open}
        onOpenChange={setOpen}
        clips={clips as AudioClip[]}
      />
    </>
  )
}

function isVideoTask(log: TaskLog) {
  return (
    log.action === TASK_ACTIONS.GENERATE ||
    log.action === TASK_ACTIONS.TEXT_GENERATE ||
    log.action === TASK_ACTIONS.FIRST_TAIL_GENERATE ||
    log.action === TASK_ACTIONS.REFERENCE_GENERATE ||
    log.action === TASK_ACTIONS.REMIX_GENERATE
  )
}

export function useTaskLogsColumns(isAdmin: boolean): ColumnDef<TaskLog>[] {
  const { t } = useTranslation()
  const columns: ColumnDef<TaskLog>[] = [
    {
      accessorKey: 'submit_time',
      header: t('Submit Time'),
      cell: ({ row }) => {
        const submitTime = row.getValue('submit_time') as number
        return submitTime ? (
          <span className='block truncate font-mono text-xs tabular-nums'>
            {formatTimestampToDate(submitTime, 'seconds')}
          </span>
        ) : (
          <span className='text-muted-foreground/60 text-xs'>-</span>
        )
      },
      size: 152,
    },
    {
      accessorKey: 'finish_time',
      header: t('End Time'),
      cell: ({ row }) => {
        const finishTime = row.getValue('finish_time') as number
        return finishTime ? (
          <span className='block truncate font-mono text-xs tabular-nums'>
            {formatTimestampToDate(finishTime, 'seconds')}
          </span>
        ) : (
          <span className='text-muted-foreground/60 text-xs'>-</span>
        )
      },
      size: 152,
    },
  ]

  if (isAdmin) {
    columns.push(
      {
        ...createChannelColumn<TaskLog>({ headerLabel: t('Channel') }),
        size: 120,
      },
      {
        id: 'user',
        header: t('User'),
        accessorFn: (row) => row.username || row.user_id,
        cell: function UserCell({ row }) {
          const { sensitiveVisible, setSelectedUserId, setUserInfoDialogOpen } =
            useUsageLogsContext()
          const log = row.original
          const displayName = log.username || String(log.user_id || '?')

          return (
            <button
              type='button'
              className='flex items-center gap-1.5 text-left'
              onClick={(e) => {
                e.stopPropagation()
                setSelectedUserId(log.user_id)
                setUserInfoDialogOpen(true)
              }}
            >
              <Avatar className='ring-border/60 size-6 ring-1 max-sm:hidden'>
                <AvatarFallback
                  className={cn(
                    'text-[11px] font-semibold',
                    !sensitiveVisible && 'bg-muted text-muted-foreground'
                  )}
                  style={
                    sensitiveVisible
                      ? getUserAvatarStyle(displayName)
                      : undefined
                  }
                >
                  {sensitiveVisible ? getUserAvatarFallback(displayName) : '•'}
                </AvatarFallback>
              </Avatar>
              <span className='text-muted-foreground truncate text-sm hover:underline'>
                {sensitiveVisible ? displayName : '••••'}
              </span>
            </button>
          )
        },
        size: 140,
      }
    )
  }

  columns.push(
    {
      accessorKey: 'task_id',
      header: t('Task ID'),
      cell: ({ row }) => {
        const taskId = row.getValue('task_id') as string
        if (!taskId) {
          return <span className='text-muted-foreground/60 text-xs'>-</span>
        }
        return (
          <div className='flex min-w-0 flex-col gap-0.5'>
            <StatusBadge
              label={taskId}
              copyText={taskId}
              variant='neutral'
              size='sm'
              className='border-border/60 bg-muted/30 !text-foreground max-w-full truncate rounded-md border px-1.5 py-0.5 font-mono'
            />
          </div>
        )
      },
      size: 210,
      meta: { mobileTitle: true },
    },
    {
      accessorKey: 'group',
      header: t('Group'),
      cell: ({ row }) => (
        <span
          className='block min-w-0 truncate text-xs'
          title={row.original.group}
        >
          {row.original.group || '-'}
        </span>
      ),
      size: 120,
    },
    {
      accessorKey: 'platform',
      header: t('Platform'),
      cell: ({ row }) => (
        <StatusBadge
          label={t(row.original.platform_name || row.original.platform)}
          variant='neutral'
          size='sm'
          copyable={false}
          className='-ml-1.5'
        />
      ),
      size: 90,
    },
    {
      accessorKey: 'action',
      header: t('Type'),
      cell: ({ row }) => (
        <StatusBadge
          label={t(taskActionMapper.getLabel(row.original.action))}
          variant={taskActionMapper.getVariant(row.original.action)}
          size='sm'
          copyable={false}
          className='-ml-1.5'
        />
      ),
      size: 110,
    },
    {
      id: 'model',
      header: t('Model'),
      accessorFn: (row) =>
        row.properties?.origin_model_name ||
        row.properties?.upstream_model_name ||
        '',
      cell: ({ row }) => {
        const model =
          row.original.properties?.origin_model_name ||
          row.original.properties?.upstream_model_name
        return model ? (
          <span
            className='block min-w-0 truncate font-mono text-xs'
            title={model}
          >
            {model}
          </span>
        ) : (
          <span className='text-muted-foreground/60 text-xs'>-</span>
        )
      },
      size: 260,
    },
    {
      ...createDurationColumn<TaskLog>({
        submitTimeKey: 'submit_time',
        finishTimeKey: 'finish_time',
        unit: 'seconds',
        headerLabel: t('Duration'),
        warningThresholdSec: 300,
      }),
      size: 90,
    },
    {
      accessorKey: 'status',
      header: t('Status'),
      cell: ({ row }) => {
        const status = row.getValue('status') as string
        return (
          <StatusBadge
            label={t(taskStatusMapper.getLabel(status, status || 'Submitting'))}
            variant={taskStatusMapper.getVariant(status)}
            size='sm'
            copyable={false}
            className='-ml-1.5'
          />
        )
      },
      size: 80,
    },
    {
      ...createProgressColumn<TaskLog>({ headerLabel: t('Progress') }),
      size: 145,
    },
    {
      id: 'preview',
      header: t('Preview'),
      cell: function PreviewCell({ row }) {
        const log = row.original
        const [previewOpen, setPreviewOpen] = useState(false)

        const isSunoSuccess =
          log.platform === 'suno' && log.status === TASK_STATUS.SUCCESS
        if (isSunoSuccess) {
          const data = parseTaskData(log.data)
          if (
            data.some(
              (c) =>
                c &&
                typeof c === 'object' &&
                (c as Record<string, unknown>).audio_url
            )
          ) {
            return <AudioPreviewCell log={log} />
          }
        }

        if (log.status === TASK_STATUS.SUCCESS && isVideoTask(log)) {
          return (
            <>
              <Button
                type='button'
                variant='outline'
                size='xs'
                onClick={() => setPreviewOpen(true)}
              >
                <Eye />
                {t('Preview and download')}
              </Button>
              <VideoPreviewDialog
                taskId={log.task_id}
                open={previewOpen}
                onOpenChange={setPreviewOpen}
              />
            </>
          )
        }

        return <span className='text-muted-foreground/60 text-xs'>-</span>
      },
      size: 130,
    },
    {
      accessorKey: 'fail_reason',
      header: t('Details'),
      cell: function DetailsCell({ row }) {
        const log = row.original
        const failReason = row.getValue('fail_reason') as string
        const [detailsOpen, setDetailsOpen] = useState(false)
        const [errorOpen, setErrorOpen] = useState(false)

        return (
          <>
            <div className='flex min-w-0 items-center gap-1.5'>
              <Button
                type='button'
                variant='ghost'
                size='xs'
                onClick={() => setDetailsOpen(true)}
              >
                <Info />
                {t('View')}
              </Button>
              {failReason ? (
                <button
                  type='button'
                  className='max-w-32 truncate text-left text-xs text-red-600 hover:underline dark:text-red-400'
                  onClick={() => setErrorOpen(true)}
                  title={t('Click to view full error message')}
                >
                  {failReason}
                </button>
              ) : null}
            </div>
            <TaskDetailsDialog
              log={log}
              open={detailsOpen}
              onOpenChange={setDetailsOpen}
            />
            {failReason ? (
              <FailReasonDialog
                failReason={failReason}
                open={errorOpen}
                onOpenChange={setErrorOpen}
              />
            ) : null}
          </>
        )
      },
      size: 150,
      maxSize: 180,
    }
  )

  return columns
}
