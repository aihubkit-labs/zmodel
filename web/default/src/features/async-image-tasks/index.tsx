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
import { Eye } from 'lucide-react'
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
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
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

function statusVariant(value: string) {
  if (value === 'failed' || value === 'refunded') {
    return 'destructive' as const
  }
  if (value === 'available' || value === 'succeeded') {
    return 'secondary' as const
  }
  return 'outline' as const
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
      {root && (
        <SectionPageLayout.Actions>
          <div className='flex flex-wrap gap-2'>
            <Button
              variant='outline'
              disabled={selected.size === 0 || retryMutation.isPending}
              onClick={() => retryMutation.mutate([...selected])}
            >
              {t('Retry selected uploads')}
            </Button>
            <AlertDialog>
              <AlertDialogTrigger
                render={
                  <Button disabled={retryAllMutation.isPending}>
                    {t('Retry all failed image uploads')}
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
                  <AlertDialogAction onClick={() => retryAllMutation.mutate()}>
                    {t('Start retry')}
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>
        </SectionPageLayout.Actions>
      )}
      <SectionPageLayout.Content>
        <div className='flex h-full min-h-0 flex-col gap-4'>
          <div className='grid gap-2 md:grid-cols-5'>
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
              placeholder={t('Output availability')}
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
                  <TableHead>{t('Task ID')}</TableHead>
                  {root && <TableHead>{t('User')}</TableHead>}
                  <TableHead>{t('Model')}</TableHead>
                  <TableHead>{t('Generation status')}</TableHead>
                  <TableHead>{t('Output availability')}</TableHead>
                  <TableHead>{t('Billing status')}</TableHead>
                  <TableHead>{t('Objects')}</TableHead>
                  <TableHead>{t('Attempts')}</TableHead>
                  <TableHead>{t('Staging integrity')}</TableHead>
                  <TableHead>{t('Error')}</TableHead>
                  <TableHead>{t('Created at')}</TableHead>
                  <TableHead>{t('Details')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(tasksQuery.data?.items ?? []).map((task) => {
                  const retryable = retryableRows.some(
                    (item) => item.task_id === task.task_id
                  )
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
                      <TableCell className='max-w-52 truncate font-mono text-xs'>
                        {task.task_id}
                      </TableCell>
                      {root && <TableCell>{task.user_id}</TableCell>}
                      <TableCell>{task.model}</TableCell>
                      <TableCell>
                        <Badge variant={statusVariant(task.status)}>
                          {t(task.status)}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Badge
                          variant={statusVariant(task.output_availability)}
                        >
                          {t(task.output_availability)}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant={statusVariant(task.billing_status)}>
                          {t(task.billing_status)}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        {task.object_available_count}/{task.object_total_count}
                      </TableCell>
                      <TableCell>{task.archive_attempts}</TableCell>
                      <TableCell>{t(task.staging_integrity)}</TableCell>
                      <TableCell className='max-w-64 truncate'>
                        {task.error || '-'}
                      </TableCell>
                      <TableCell>
                        {dayjs
                          .unix(task.created_at)
                          .format('YYYY-MM-DD HH:mm:ss')}
                      </TableCell>
                      <TableCell>
                        <Button
                          type='button'
                          variant='ghost'
                          size='xs'
                          onClick={() => setDetailsTask(task)}
                        >
                          <Eye />
                          {t('Preview and download')}
                        </Button>
                      </TableCell>
                    </TableRow>
                  )
                })}
                {!tasksQuery.isLoading &&
                  (tasksQuery.data?.items.length ?? 0) === 0 && (
                    <TableRow>
                      <TableCell
                        colSpan={root ? 13 : 11}
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
      </SectionPageLayout.Content>
      <AsyncImageTaskDetailsDialog
        task={detailsTask}
        root={root}
        onClose={() => setDetailsTask(null)}
      />
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
      <SelectTrigger>
        <SelectValue placeholder={props.placeholder} />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value='all'>{t('All')}</SelectItem>
        {props.options
          .filter((option) => option !== '')
          .map((option) => (
            <SelectItem key={option} value={option}>
              {t(option)}
            </SelectItem>
          ))}
      </SelectContent>
    </Select>
  )
}
