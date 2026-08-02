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
import { getRouteApi } from '@tanstack/react-router'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  DISABLED_ROW_DESKTOP,
  DISABLED_ROW_MOBILE,
  DataTablePage,
  useDataTable,
} from '@/components/data-table'
import { DateTimeRangePicker } from '@/components/date-time-range-picker'
import { Skeleton } from '@/components/ui/skeleton'
import { useMediaQuery } from '@/hooks'
import { useTableUrlState } from '@/hooks/use-table-url-state'
import { formatQuota } from '@/lib/format'

import { getRedemptions, searchRedemptions } from '../api'
import {
  ERROR_MESSAGES,
  REDEMPTION_STATUS,
  getRedemptionStatusOptions,
} from '../constants'
import { isRedemptionExpired } from '../lib'
import type { Redemption } from '../types'
import { DataTableBulkActions } from './data-table-bulk-actions'
import { useRedemptionsColumns } from './redemptions-columns'
import { RedemptionsMobileList } from './redemptions-mobile-list'
import { useRedemptions } from './redemptions-provider'

const route = getRouteApi('/_authenticated/redemption-codes/')

function isDisabledRedemptionRow(redemption: Redemption) {
  return (
    redemption.status !== REDEMPTION_STATUS.ENABLED ||
    isRedemptionExpired(redemption.expired_time, redemption.status)
  )
}

export function RedemptionsTable() {
  const { t } = useTranslation()
  const columns = useRedemptionsColumns()
  const { refreshTrigger } = useRedemptions()
  const isMobile = useMediaQuery('(max-width: 640px)')
  const search = route.useSearch()
  const navigate = route.useNavigate()

  const {
    globalFilter,
    onGlobalFilterChange,
    columnFilters,
    onColumnFiltersChange,
    pagination,
    onPaginationChange,
    ensurePageInRange,
  } = useTableUrlState({
    search,
    navigate,
    pagination: { defaultPage: 1, defaultPageSize: isMobile ? 10 : 20 },
    globalFilter: { enabled: true, key: 'filter' },
    columnFilters: [{ columnId: 'status', searchKey: 'status', type: 'array' }],
  })
  const statusFilter =
    (columnFilters.find((filter) => filter.id === 'status')?.value as
      | string[]
      | undefined) ?? []
  const statusFilterValue = statusFilter[0] ?? ''
  const redeemedStart = search.startTime
    ? new Date(search.startTime * 1000)
    : undefined
  const redeemedEnd = search.endTime
    ? new Date(search.endTime * 1000)
    : undefined
  const hasRedeemedTimeFilter = Boolean(search.startTime || search.endTime)

  // Fetch data with React Query
  const { data, isLoading, isFetching, isPlaceholderData } = useQuery({
    queryKey: [
      'redemptions',
      pagination.pageIndex + 1,
      pagination.pageSize,
      globalFilter,
      statusFilterValue,
      search.startTime,
      search.endTime,
      refreshTrigger,
    ],
    queryFn: async () => {
      const hasFilter = globalFilter?.trim()
      const hasStatusFilter = statusFilterValue !== ''
      const hasSearchFilters =
        hasFilter || hasStatusFilter || hasRedeemedTimeFilter
      const params = {
        p: pagination.pageIndex + 1,
        page_size: pagination.pageSize,
      }

      const result = hasSearchFilters
        ? await searchRedemptions({
            ...params,
            keyword: globalFilter,
            status: statusFilterValue,
            start_timestamp: search.startTime,
            end_timestamp: search.endTime,
          })
        : await getRedemptions(params)

      if (!result.success) {
        toast.error(
          result.message ||
            t(
              hasSearchFilters
                ? ERROR_MESSAGES.SEARCH_FAILED
                : ERROR_MESSAGES.LOAD_FAILED
            )
        )
        return { items: [], total: 0, totalQuota: 0 }
      }

      return {
        items: result.data?.items || [],
        total: result.data?.total || 0,
        totalQuota: result.data?.total_quota || 0,
      }
    },
    placeholderData: (previousData) => previousData,
  })

  const redemptions = data?.items || []

  const { table } = useDataTable({
    data: redemptions,
    columns,
    enableRowSelection: true,
    columnFilters,
    globalFilter,
    pagination,
    globalFilterFn: (row, _columnId, filterValue) => {
      const name = String(row.getValue('name')).toLowerCase()
      const id = String(row.getValue('id'))
      const searchValue = String(filterValue).toLowerCase()

      return name.includes(searchValue) || id.includes(searchValue)
    },
    onPaginationChange,
    onGlobalFilterChange,
    onColumnFiltersChange,
    manualPagination: true,
    manualFiltering: true,
    totalCount: data?.total || 0,
    ensurePageInRange,
  })

  const redemptionStatusOptions = useMemo(
    () => getRedemptionStatusOptions(t),
    [t]
  )

  const handleRedeemedTimeChange = (range: { start?: Date; end?: Date }) => {
    navigate({
      search: (previous) => ({
        ...previous,
        page: undefined,
        startTime: range.start
          ? Math.floor(range.start.getTime() / 1000)
          : undefined,
        endTime: range.end ? Math.floor(range.end.getTime() / 1000) : undefined,
      }),
    })
  }

  return (
    <DataTablePage
      table={table}
      columns={columns}
      isLoading={isLoading}
      isFetching={isFetching}
      emptyTitle={t('No Redemption Codes Found')}
      emptyDescription={t(
        'No redemption codes available. Create your first redemption code to get started.'
      )}
      skeletonKeyPrefix='redemptions-skeleton'
      applyHeaderSize
      toolbarProps={{
        searchPlaceholder: t('Filter by name or ID...'),
        additionalSearch: (
          <DateTimeRangePicker
            start={redeemedStart}
            end={redeemedEnd}
            onChange={handleRedeemedTimeChange}
            monthOptionsCount={36}
            className='sm:w-[300px]'
          />
        ),
        hasAdditionalFilters: hasRedeemedTimeFilter,
        onReset: () => {
          navigate({
            search: (previous) => ({
              ...previous,
              page: undefined,
              startTime: undefined,
              endTime: undefined,
            }),
          })
        },
        filters: [
          {
            columnId: 'status',
            title: t('Status'),
            options: redemptionStatusOptions,
            singleSelect: true,
          },
        ],
        leftActions: (
          <div className='flex min-h-8 items-baseline gap-2'>
            <span className='text-muted-foreground text-sm'>
              {t('Total Quota')}
            </span>
            {isLoading || isPlaceholderData ? (
              <Skeleton className='h-6 w-24' />
            ) : (
              <span className='text-lg font-semibold tabular-nums'>
                {formatQuota(data?.totalQuota ?? 0)}
              </span>
            )}
          </div>
        ),
      }}
      mobile={<RedemptionsMobileList table={table} isLoading={isLoading} />}
      getRowClassName={(row, { isMobile }) => {
        if (!isDisabledRedemptionRow(row.original)) return undefined
        return isMobile ? DISABLED_ROW_MOBILE : DISABLED_ROW_DESKTOP
      }}
      bulkActions={<DataTableBulkActions table={table} />}
    />
  )
}
