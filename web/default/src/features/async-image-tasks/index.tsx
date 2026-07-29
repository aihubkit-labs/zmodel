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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import dayjs from 'dayjs'
import { FileText } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import {
  getAsyncImageTasks,
  retryAllFailedAsyncImageTasks,
  retryAsyncImageTasks,
  type AsyncImageTaskFilters,
} from './api'
import { AsyncImageTaskDetailsDialog } from './components/task-details-dialog'
import type { AsyncImageTask } from './types'

const pageSize = 20
const statusOptions = ['', 'queued', 'running', 'succeeded', 'failed']
const outputOptions = [
  '',
  'pending',
  'archiving',
  'available',
  'expired',
  'failed',
]
const billingOptions = ['', 'reserved', 'settled', 'refunded']

type HideableColumnId =
  | 'submit_time'
  | 'end_time'
  | 'channel'
  | 'user'
  | 'group'
  | 'platform'
  | 'model'
  | 'duration'
  | 'generation_status'
  | 'output_availability'
  | 'billing_status'
  | 'objects'
  | 'attempts'
  | 'staging_integrity'
  | 'error'

type ColumnVisibility = Record<HideableColumnId, boolean>
type ColumnVisibilityScope = 'admin' | 'user'

const columnVisibilityStorageKeys: Record<ColumnVisibilityScope, string> = {
  admin: 'async-image-tasks:admin:column-visibility:v2',
  user: 'async-image-tasks:user:column-visibility:v2',
}

const defaultColumnVisibility: ColumnVisibility = {
  submit_time: true,
  end_time: true,
  channel: true,
  user: true,
  group: true,
  platform: false,
  model: true,
  duration: true,
  generation_status: true,
  output_availability: true,
  billing_status: false,
  objects: false,
  attempts: false,
  staging_integrity: false,
  error: false,
}

const hideableColumns: Array<{
  id: HideableColumnId
  label: string
  rootOnly?: boolean
}> = [
  { id: 'submit_time', label: 'Submit Time' },
  { id: 'end_time', label: 'End Time' },
  { id: 'user', label: 'User', rootOnly: true },
  { id: 'group', label: 'Group' },
  { id: 'channel', label: 'Channel', rootOnly: true },
  { id: 'platform', label: 'Platform', rootOnly: true },
  { id: 'model', label: 'Model' },
  { id: 'duration', label: 'Duration' },
  { id: 'generation_status', label: 'Generation status' },
  { id: 'output_availability', label: 'Upload status' },
  { id: 'billing_status', label: 'Billing status' },
  { id: 'objects', label: 'Objects' },
  { id: 'attempts', label: 'Attempts' },
  { id: 'staging_integrity', label: 'Staging integrity' },
  { id: 'error', label: 'Error' },
]

function readColumnVisibility(scope: ColumnVisibilityScope): ColumnVisibility {
  if (typeof window === 'undefined') return defaultColumnVisibility

  try {
    const stored = window.localStorage.getItem(
      columnVisibilityStorageKeys[scope]
    )
    if (!stored) return defaultColumnVisibility
    const parsed = JSON.parse(stored) as Partial<ColumnVisibility>
    return { ...defaultColumnVisibility, ...parsed }
  } catch {
    return defaultColumnVisibility
  }
}

function statusVariant(value: string) {
  if (value === 'failed' || value === 'refunded') {
    return 'destructive' as const
  }
  if (value === 'available' || value === 'succeeded') {
    return 'secondary' as const
  }
  return 'outline' as const
}

function formatTaskTime(timestamp: number) {
  return timestamp ? dayjs.unix(timestamp).format('YYYY-MM-DD HH:mm:ss') : '-'
}

export function AsyncImageTasksPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const userRole = useAuthStore((state) => state.auth.user?.role)
  const root = userRole === ROLE.SUPER_ADMIN
  const [page, setPage] = useState(1)
  const [taskId, setTaskId] = useState('')
  const [model, setModel] = useState('')
  const [status, setStatus] = useState('')
  const [outputAvailability, setOutputAvailability] = useState('')
  const [billingStatus, setBillingStatus] = useState('')
  const [selected, setSelected] = useState<Set<string>>(() => new Set())
  const [detailsTask, setDetailsTask] = useState<AsyncImageTask | null>(null)
  const [columnVisibilityByScope, setColumnVisibilityByScope] = useState<
    Record<ColumnVisibilityScope, ColumnVisibility>
  >(() => ({
    admin: readColumnVisibility('admin'),
    user: readColumnVisibility('user'),
  }))

  const visibilityScope: ColumnVisibilityScope = root ? 'admin' : 'user'
  const columnVisibility = columnVisibilityByScope[visibilityScope]
  const availableHideableColumns = hideableColumns.filter(
    (column) => root || !column.rootOnly
  )
  const visibleColumnCount =
    (root ? 1 : 0) +
    2 +
    availableHideableColumns.filter((column) => columnVisibility[column.id])
      .length

  const setColumnVisible = (columnId: HideableColumnId, visible: boolean) => {
    const nextVisibility = {
      ...columnVisibility,
      [columnId]: visible,
    }
    setColumnVisibilityByScope((current) => ({
      ...current,
      [visibilityScope]: nextVisibility,
    }))
    try {
      window.localStorage.setItem(
        columnVisibilityStorageKeys[visibilityScope],
        JSON.stringify(nextVisibility)
      )
    } catch {
      // Column selection still works for this session when storage is blocked.
    }
  }

  const filters: AsyncImageTaskFilters = {
    page,
    page_size: pageSize,
    task_id: taskId || undefined,
    model: model || undefined,
    status: status || undefined,
    output_availability: outputAvailability || undefined,
    billing_status: billingStatus || undefined,
  }
  const tasksQuery = useQuery({
    queryKey: ['async-image-tasks', root, filters],
    queryFn: () => getAsyncImageTasks(filters, root),
    refetchInterval: 15_000,
  })

  const refresh = async () => {
    setSelected(new Set())
    await queryClient.invalidateQueries({ queryKey: ['async-image-tasks'] })
  }
  const retryMutation = useMutation({
    mutationFn: retryAsyncImageTasks,
    onSuccess: async (result) => {
      toast.success(
        t('Accepted {{accepted}} tasks; skipped {{skipped}} tasks', {
          accepted: result.accepted_count,
          skipped: result.skipped_count,
        })
      )
      await refresh()
    },
    onError: () => toast.error(t('Failed to retry image uploads')),
  })
  const retryAllMutation = useMutation({
    mutationFn: retryAllFailedAsyncImageTasks,
    onSuccess: async (result) => {
      toast.success(
        t('Bulk retry operation started: {{operationId}}', {
          operationId: result.operation_id,
        })
      )
      await refresh()
    },
    onError: () => toast.error(t('Failed to start bulk retry')),
  })

  const retryableRows = (tasksQuery.data?.items ?? []).filter(
    (task) =>
      task.status === 'succeeded' &&
      task.billing_status === 'settled' &&
      task.output_availability === 'failed' &&
      task.staging_integrity === 'available'
  )
  const allRetryableSelected =
    retryableRows.length > 0 &&
    retryableRows.every((task) => selected.has(task.task_id))
  const totalPages = Math.max(
    1,
    Math.ceil((tasksQuery.data?.total ?? 0) / pageSize)
  )

  const toggleAll = (checked: boolean) => {
    if (!checked) {
      setSelected(new Set())
      return
    }
    setSelected(new Set(retryableRows.map((task) => task.task_id)))
  }

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>
        {t('Async Image Tasks')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <div className='flex flex-wrap gap-2'>
          {root && (
            <>
              <Button
                variant='outline'
                disabled={selected.size === 0 || retryMutation.isPending}
                onClick={() => retryMutation.mutate([...selected])}
              >
                {t('Retry selected')}
              </Button>
              <AlertDialog>
                <AlertDialogTrigger
                  render={
                    <Button disabled={retryAllMutation.isPending}>
                      {t('Retry all failed')}
                    </Button>
                  }
                />
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>
                      {t('Retry all failed uploads?')}
                    </AlertDialogTitle>
                    <AlertDialogDescription>
                      {t(
                        'Only persistent staged files are uploaded again. Image generation and billing are not repeated.'
                      )}
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>{t('Cancel')}</AlertDialogCancel>
                    <AlertDialogAction
                      onClick={() => retryAllMutation.mutate()}
                    >
                      {t('Start retry')}
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </>
          )}
          <DropdownMenu modal={false}>
            <DropdownMenuTrigger
              render={
                <Button
                  variant='outline'
                  className='shrink-0'
                  aria-label={t('View')}
                />
              }
            >
              {t('View')}
            </DropdownMenuTrigger>
            <DropdownMenuContent align='end' className='w-[180px]'>
              <DropdownMenuGroup>
                <DropdownMenuLabel>{t('Toggle columns')}</DropdownMenuLabel>
                {availableHideableColumns.map((column) => (
                  <DropdownMenuCheckboxItem
                    key={column.id}
                    checked={columnVisibility[column.id]}
                    onCheckedChange={(checked) =>
                      setColumnVisible(column.id, checked)
                    }
                  >
                    {t(column.label)}
                  </DropdownMenuCheckboxItem>
                ))}
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='flex h-full min-h-0 flex-col gap-4'>
          <div className='grid grid-cols-1 gap-2 md:grid-cols-2 xl:grid-cols-5'>
            <Input
              value={taskId}
              onChange={(event) => {
                setTaskId(event.target.value)
                setPage(1)
              }}
              placeholder={t('Task ID')}
            />
            <Input
              value={model}
              onChange={(event) => {
                setModel(event.target.value)
                setPage(1)
              }}
              placeholder={t('Model')}
            />
            <StatusSelect
              value={status}
              options={statusOptions}
              placeholder={t('Generation status')}
              onChange={(value) => {
                setStatus(value)
                setPage(1)
              }}
            />
            <StatusSelect
              value={outputAvailability}
              options={outputOptions}
              placeholder={t('Upload status')}
              onChange={(value) => {
                setOutputAvailability(value)
                setPage(1)
              }}
            />
            <StatusSelect
              value={billingStatus}
              options={billingOptions}
              placeholder={t('Billing status')}
              onChange={(value) => {
                setBillingStatus(value)
                setPage(1)
              }}
            />
          </div>

          <div className='min-h-0 flex-1 overflow-auto rounded-xl border'>
            <Table>
              <TableHeader>
                <TableRow>
                  {root && (
                    <TableHead className='w-10'>
                      <Checkbox
                        checked={allRetryableSelected}
                        onCheckedChange={(checked) =>
                          toggleAll(checked === true)
                        }
                        aria-label={t('Select retryable tasks')}
                      />
                    </TableHead>
                  )}
                  {columnVisibility.submit_time && (
                    <TableHead>{t('Submit Time')}</TableHead>
                  )}
                  {columnVisibility.end_time && (
                    <TableHead>{t('End Time')}</TableHead>
                  )}
                  {root && columnVisibility.user && (
                    <TableHead>{t('User')}</TableHead>
                  )}
                  <TableHead>{t('Task ID')}</TableHead>
                  {columnVisibility.group && (
                    <TableHead>{t('Group')}</TableHead>
                  )}
                  {root && columnVisibility.channel && (
                    <TableHead>{t('Channel')}</TableHead>
                  )}
                  {root && columnVisibility.platform && (
                    <TableHead>{t('Platform')}</TableHead>
                  )}
                  {columnVisibility.model && (
                    <TableHead>{t('Model')}</TableHead>
                  )}
                  {columnVisibility.duration && (
                    <TableHead>{t('Duration')}</TableHead>
                  )}
                  {columnVisibility.generation_status && (
                    <TableHead>{t('Generation status')}</TableHead>
                  )}
                  {columnVisibility.output_availability && (
                    <TableHead>{t('Upload status')}</TableHead>
                  )}
                  {columnVisibility.billing_status && (
                    <TableHead>{t('Billing status')}</TableHead>
                  )}
                  {columnVisibility.objects && (
                    <TableHead>{t('Objects')}</TableHead>
                  )}
                  {columnVisibility.attempts && (
                    <TableHead>{t('Attempts')}</TableHead>
                  )}
                  {columnVisibility.staging_integrity && (
                    <TableHead>{t('Staging integrity')}</TableHead>
                  )}
                  {columnVisibility.error && (
                    <TableHead>{t('Error')}</TableHead>
                  )}
                  <TableHead>{t('Details')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(tasksQuery.data?.items ?? []).map((task) => {
                  const retryable = retryableRows.some(
                    (item) => item.task_id === task.task_id
                  )
                  const durationSeconds =
                    task.completed_at > task.created_at
                      ? task.completed_at - task.created_at
                      : null
                  const channelLabel = task.channel_name || '-'
                  return (
                    <TableRow key={task.task_id}>
                      {root && (
                        <TableCell>
                          <Checkbox
                            disabled={!retryable}
                            checked={selected.has(task.task_id)}
                            onCheckedChange={(checked) => {
                              setSelected((current) => {
                                const next = new Set(current)
                                if (checked === true) next.add(task.task_id)
                                else next.delete(task.task_id)
                                return next
                              })
                            }}
                            aria-label={t('Select task {{taskId}}', {
                              taskId: task.task_id,
                            })}
                          />
                        </TableCell>
                      )}
                      {columnVisibility.submit_time && (
                        <TableCell className='font-mono text-xs whitespace-nowrap tabular-nums'>
                          {formatTaskTime(task.created_at)}
                        </TableCell>
                      )}
                      {columnVisibility.end_time && (
                        <TableCell className='font-mono text-xs whitespace-nowrap tabular-nums'>
                          {formatTaskTime(task.completed_at)}
                        </TableCell>
                      )}
                      {root && columnVisibility.user && (
                        <TableCell className='max-w-40 truncate'>
                          {task.username || `${t('User ID')}: ${task.user_id}`}
                        </TableCell>
                      )}
                      <TableCell
                        className='font-mono text-xs whitespace-nowrap'
                        title={task.task_id}
                      >
                        {task.task_id}
                      </TableCell>
                      {columnVisibility.group && (
                        <TableCell className='max-w-32 truncate'>
                          {task.using_group || '-'}
                        </TableCell>
                      )}
                      {root && columnVisibility.channel && (
                        <TableCell className='max-w-44 truncate'>
                          {channelLabel}
                        </TableCell>
                      )}
                      {root && columnVisibility.platform && (
                        <TableCell>
                          {task.platform ? (
                            <Badge variant='outline'>{task.platform}</Badge>
                          ) : (
                            '-'
                          )}
                        </TableCell>
                      )}
                      {columnVisibility.model && (
                        <TableCell className='max-w-52 truncate'>
                          {task.model}
                        </TableCell>
                      )}
                      {columnVisibility.duration && (
                        <TableCell className='font-mono text-xs whitespace-nowrap tabular-nums'>
                          {durationSeconds === null
                            ? '-'
                            : `${durationSeconds.toFixed(1)}s`}
                        </TableCell>
                      )}
                      {columnVisibility.generation_status && (
                        <TableCell>
                          <Badge variant={statusVariant(task.status)}>
                            {t(task.status)}
                          </Badge>
                        </TableCell>
                      )}
                      {columnVisibility.output_availability && (
                        <TableCell>
                          <Badge
                            variant={statusVariant(task.output_availability)}
                          >
                            {t(
                              task.output_availability === 'available'
                                ? 'Uploaded'
                                : task.output_availability
                            )}
                          </Badge>
                        </TableCell>
                      )}
                      {columnVisibility.billing_status && (
                        <TableCell>
                          <Badge variant={statusVariant(task.billing_status)}>
                            {t(task.billing_status)}
                          </Badge>
                        </TableCell>
                      )}
                      {columnVisibility.objects && (
                        <TableCell>
                          {task.object_available_count}/
                          {task.object_total_count}
                        </TableCell>
                      )}
                      {columnVisibility.attempts && (
                        <TableCell>{task.archive_attempts}</TableCell>
                      )}
                      {columnVisibility.staging_integrity && (
                        <TableCell>{t(task.staging_integrity)}</TableCell>
                      )}
                      {columnVisibility.error && (
                        <TableCell className='max-w-64 truncate'>
                          {task.error || '-'}
                        </TableCell>
                      )}
                      <TableCell>
                        <Button
                          type='button'
                          variant='ghost'
                          size='xs'
                          onClick={() => setDetailsTask(task)}
                        >
                          <FileText />
                          {t('Details')}
                        </Button>
                      </TableCell>
                    </TableRow>
                  )
                })}
                {!tasksQuery.isLoading &&
                  (tasksQuery.data?.items.length ?? 0) === 0 && (
                    <TableRow>
                      <TableCell
                        colSpan={visibleColumnCount}
                        className='text-muted-foreground h-32 text-center'
                      >
                        {t('No async image tasks found')}
                      </TableCell>
                    </TableRow>
                  )}
              </TableBody>
            </Table>
          </div>

          <div className='flex items-center justify-between gap-3'>
            <span className='text-muted-foreground text-sm'>
              {t('{{count}} tasks', { count: tasksQuery.data?.total ?? 0 })}
            </span>
            <div className='flex items-center gap-2'>
              <Button
                variant='outline'
                size='sm'
                disabled={page <= 1}
                onClick={() => setPage((current) => current - 1)}
              >
                {t('Previous')}
              </Button>
              <span className='text-sm'>
                {page}/{totalPages}
              </span>
              <Button
                variant='outline'
                size='sm'
                disabled={page >= totalPages}
                onClick={() => setPage((current) => current + 1)}
              >
                {t('Next')}
              </Button>
            </div>
          </div>
        </div>
        <AsyncImageTaskDetailsDialog
          task={detailsTask}
          root={root}
          onClose={() => setDetailsTask(null)}
        />
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

type StatusSelectProps = {
  value: string
  options: string[]
  placeholder: string
  onChange: (value: string) => void
}

function StatusSelect(props: StatusSelectProps) {
  const { t } = useTranslation()
  return (
    <Select
      value={props.value || 'all'}
      onValueChange={(value) => {
        if (value !== null) props.onChange(value === 'all' ? '' : value)
      }}
    >
      <SelectTrigger className='w-full'>
        <SelectValue placeholder={props.placeholder} />
      </SelectTrigger>
      <SelectContent>
        <SelectGroup>
          <SelectItem value='all'>{t('All')}</SelectItem>
          {props.options
            .filter((option) => option !== '')
            .map((option) => (
              <SelectItem key={option} value={option}>
                {t(option === 'available' ? 'Uploaded' : option)}
              </SelectItem>
            ))}
        </SelectGroup>
      </SelectContent>
    </Select>
  )
}
