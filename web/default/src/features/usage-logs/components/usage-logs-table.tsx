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
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { getRouteApi } from '@tanstack/react-router'
import type { ColumnDef } from '@tanstack/react-table'
import { useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  DataTablePage,
  DataTableRow,
  useDataTable,
} from '@/components/data-table'
import { useMediaQuery } from '@/hooks'
import { useTableUrlState } from '@/hooks/use-table-url-state'
import { cn } from '@/lib/utils'

import {
  DEFAULT_LOGS_DATA,
  LOG_TYPE_ALL_VALUE,
  LOG_TYPE_ENUM,
} from '../constants'
import { useColumnsByCategory } from '../lib/columns'
import { parseLogOther } from '../lib/format'
import { fetchLogsByCategory, mergeTaskLogProgress } from '../lib/utils'
import { canUploadVideoTaskToS3 } from '../lib/video-task'
import type { LogCategory, TaskLog } from '../types'
import { CommonLogsFilterBar } from './common-logs-filter-bar'
import { TaskLogsBulkActions } from './task-logs-bulk-actions'
import { TaskLogsFilterBar } from './task-logs-filter-bar'
import { UsageLogsMobileList } from './usage-logs-mobile-card'
import { useLogsViewScope } from './usage-logs-provider'

const route = getRouteApi('/_authenticated/usage-logs/$section')

const TASK_LOG_AUTO_REFRESH_INTERVAL_MS = 5000

function isTaskLogActive(log: Record<string, unknown>): boolean {
  const status = typeof log.status === 'string' ? log.status : ''
  if (status === 'SUCCESS' || status === 'FAILURE') {
    return false
  }

  const progress = typeof log.progress === 'string' ? log.progress : ''
  const numericProgress = Number.parseFloat(progress.replace('%', ''))
  if (Number.isFinite(numericProgress)) {
    return numericProgress < 100
  }

  return (
    status === '' ||
    status === 'NOT_START' ||
    status === 'SUBMITTED' ||
    status === 'IN_PROGRESS' ||
    status === 'QUEUED'
  )
}

const logTypeRowTint: Record<number, string> = {
  [LOG_TYPE_ENUM.ERROR]: 'bg-rose-50/40 dark:bg-rose-950/20',
  [LOG_TYPE_ENUM.REFUND]: 'bg-blue-50/30 dark:bg-blue-950/15',
}

// Warning tint for logs where a quota conversion saturated (admin-only marker).
// Takes precedence over the per-type tint since it flags a billing anomaly.
const quotaSaturationRowTint = 'bg-amber-50/60 dark:bg-amber-950/25'

function getColumnVisibilityStorageKey(
  logCategory: LogCategory,
  isAdmin: boolean
): string {
  return `usage-logs:${logCategory}:${isAdmin ? 'admin' : 'user'}:column-visibility`
}

function deserializeLogTypeFilter(value: unknown): unknown[] {
  let values: unknown[] = []
  if (Array.isArray(value)) {
    values = value
  } else if (value) {
    values = [value]
  }
  return values.filter((item) => String(item) !== LOG_TYPE_ALL_VALUE)
}

interface UsageLogsTableProps {
  logCategory: LogCategory
}

export function UsageLogsTable({ logCategory }: UsageLogsTableProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { isAdminView: isAdmin } = useLogsViewScope()
  const isMobile = useMediaQuery('(max-width: 640px)')
  const searchParams = route.useSearch()

  const {
    columnFilters,
    onColumnFiltersChange,
    pagination,
    onPaginationChange,
    ensurePageInRange,
  } = useTableUrlState({
    search: route.useSearch(),
    navigate: route.useNavigate(),
    pagination: { defaultPage: 1, defaultPageSize: isMobile ? 20 : 100 },
    globalFilter: { enabled: false },
    columnFilters: [
      {
        columnId: 'created_at',
        searchKey: 'type',
        type: 'array' as const,
        deserialize: deserializeLogTypeFilter,
      },
      { columnId: 'model_name', searchKey: 'model', type: 'string' as const },
      { columnId: 'token_name', searchKey: 'token', type: 'string' as const },
      { columnId: 'group', searchKey: 'group', type: 'string' as const },
      ...(isAdmin
        ? [
            {
              columnId: 'channel',
              searchKey: 'channel',
              type: 'string' as const,
            },
            {
              columnId: 'username',
              searchKey: 'username',
              type: 'string' as const,
            },
          ]
        : []),
    ],
  })

  const logsQueryKey = useMemo(
    () => [
      'logs',
      logCategory,
      isAdmin,
      pagination.pageIndex + 1,
      pagination.pageSize,
      columnFilters,
      searchParams,
      t,
    ],
    [
      columnFilters,
      isAdmin,
      logCategory,
      pagination.pageIndex,
      pagination.pageSize,
      searchParams,
      t,
    ]
  )

  const { data, isLoading, isFetching } = useQuery({
    queryKey: logsQueryKey,
    queryFn: async () => {
      const result = await fetchLogsByCategory({
        logCategory,
        isAdmin,
        page: pagination.pageIndex + 1,
        pageSize: pagination.pageSize,
        searchParams,
        columnFilters,
      })

      if (!result?.success) {
        toast.error(result?.message || t('Failed to load logs'))
        return DEFAULT_LOGS_DATA
      }

      return result.data || DEFAULT_LOGS_DATA
    },
    placeholderData: (previousData, previousQuery) => {
      if (previousQuery?.queryKey[1] === logCategory) {
        return previousData
      }
      return undefined
    },
  })

  const logs = data?.items || []
  const hasActiveTaskLogs =
    logCategory === 'task' &&
    logs.some((item) => isTaskLogActive(item as Record<string, unknown>))

  useEffect(() => {
    if (!hasActiveTaskLogs) {
      return undefined
    }

    let stopped = false
    const timer = window.setInterval(async () => {
      const result = await fetchLogsByCategory({
        logCategory,
        isAdmin,
        page: pagination.pageIndex + 1,
        pageSize: pagination.pageSize,
        searchParams,
        columnFilters,
      })

      const refreshedData = result.data
      if (stopped || !result?.success || !refreshedData) {
        return
      }

      queryClient.setQueryData(logsQueryKey, (currentData: typeof data) => {
        if (!currentData) {
          return currentData
        }
        return mergeTaskLogProgress(
          { ...currentData, items: currentData.items as TaskLog[] },
          { items: refreshedData.items as TaskLog[] }
        )
      })

      const refreshedItems = refreshedData.items
      if (
        !refreshedItems.some((item) =>
          isTaskLogActive(item as Record<string, unknown>)
        )
      ) {
        window.clearInterval(timer)
      }
    }, TASK_LOG_AUTO_REFRESH_INTERVAL_MS)

    return () => {
      stopped = true
      window.clearInterval(timer)
    }
  }, [
    columnFilters,
    hasActiveTaskLogs,
    isAdmin,
    logCategory,
    logsQueryKey,
    pagination.pageIndex,
    pagination.pageSize,
    queryClient,
    searchParams,
  ])

  const columns = useColumnsByCategory(logCategory, isAdmin)
  const isLoadingData = isLoading || (isFetching && !data)

  const { table } = useDataTable({
    data: logs as Record<string, unknown>[],
    columns: columns as ColumnDef<Record<string, unknown>>[],
    columnFilters,
    columnVisibilityStorageKey: getColumnVisibilityStorageKey(
      logCategory,
      isAdmin
    ),
    pagination,
    enableRowSelection:
      isAdmin && logCategory === 'task'
        ? (row) => canUploadVideoTaskToS3(row.original as unknown as TaskLog)
        : false,
    onPaginationChange,
    onColumnFiltersChange,
    manualPagination: true,
    manualFiltering: true,
    totalCount: data?.total || 0,
    ensurePageInRange,
  })

  const isCommon = logCategory === 'common'

  return (
    <DataTablePage
      table={table}
      columns={columns as ColumnDef<Record<string, unknown>>[]}
      isLoading={isLoadingData}
      isFetching={isFetching}
      emptyTitle={t('No Logs Found')}
      emptyDescription={t(
        'No usage logs available. Logs will appear here once API calls are made.'
      )}
      skeletonKeyPrefix='usage-log-skeleton'
      applyHeaderSize
      tableClassName={cn(
        '[&_[data-slot=table]]:text-[13px] [&_[data-slot=table]_td]:text-[13px] [&_[data-slot=table]_td_*]:text-[13px] [&_[data-slot=table]_th]:text-[13px] [&_[data-slot=table]_th_*]:text-[13px]'
      )}
      mobile={
        <UsageLogsMobileList
          table={table}
          isLoading={isLoadingData}
          logCategory={logCategory}
        />
      }
      toolbar={
        isCommon ? (
          <CommonLogsFilterBar table={table} />
        ) : (
          <TaskLogsFilterBar table={table} logCategory={logCategory} />
        )
      }
      bulkActions={
        isAdmin && logCategory === 'task' ? (
          <TaskLogsBulkActions table={table} />
        ) : undefined
      }
      renderRow={(row) => {
        const logType = (row.original as Record<string, unknown>).type as
          | number
          | undefined
        let tintClass =
          isCommon && logType != null ? (logTypeRowTint[logType] ?? '') : ''
        if (isCommon && isAdmin) {
          const other = parseLogOther(
            ((row.original as Record<string, unknown>).other as string) ?? ''
          )
          if (other?.admin_info?.quota_saturation) {
            tintClass = quotaSaturationRowTint
          }
        }

        return (
          <DataTableRow
            key={row.id}
            row={row}
            className={cn('transition-colors', tintClass)}
            getColumnClassName={() => (isCommon ? 'py-2' : 'py-3.5')}
          />
        )
      }}
    />
  )
}
