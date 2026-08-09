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
import type { Table } from '@tanstack/react-table'
import { Loader2, Upload } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { DataTableBulkActions } from '@/components/data-table'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import { batchVideoUpload } from '../api'
import type { TaskLog } from '../types'

interface TaskLogsBulkActionsProps {
  table: Table<Record<string, unknown>>
}

export function TaskLogsBulkActions(props: TaskLogsBulkActionsProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const selectedRows = props.table.getFilteredSelectedRowModel().rows
  const uploadMutation = useMutation({
    mutationFn: () =>
      batchVideoUpload(
        selectedRows.map((row) => (row.original as unknown as TaskLog).task_id)
      ),
    onSuccess: (result) => {
      if (result.accepted.length > 0) {
        toast.success(t('Video uploads started'))
      }
      if (result.skipped.length > 0) {
        toast.warning(
          t('{{count}} task(s) were skipped', { count: result.skipped.length })
        )
      }
      props.table.resetRowSelection()
      queryClient.invalidateQueries({ queryKey: ['logs'] })
    },
    onError: () => {
      toast.error(t('Failed to upload videos to S3'))
    },
  })

  return (
    <DataTableBulkActions table={props.table} entityName='task'>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              type='button'
              variant='outline'
              size='icon'
              className='size-8'
              disabled={uploadMutation.isPending}
              onClick={() => uploadMutation.mutate()}
              aria-label={t('Upload to S3')}
            />
          }
        >
          {uploadMutation.isPending ? (
            <Loader2 className='size-4 animate-spin' />
          ) : (
            <Upload className='size-4' />
          )}
        </TooltipTrigger>
        <TooltipContent>{t('Upload to S3')}</TooltipContent>
      </Tooltip>
    </DataTableBulkActions>
  )
}
