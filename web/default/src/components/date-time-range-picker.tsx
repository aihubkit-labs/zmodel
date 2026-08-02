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
import { CalendarDays } from 'lucide-react'
import { useId, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { toIntlLocale } from '@/i18n/languages'
import dayjs from '@/lib/dayjs'
import { cn } from '@/lib/utils'

interface DateTimeRangePickerProps {
  start?: Date
  end?: Date
  onChange: (range: { start?: Date; end?: Date }) => void
  className?: string
  monthOptionsCount?: number
}

function toInputValue(date?: Date): string {
  return date ? dayjs(date).format('YYYY-MM-DDTHH:mm') : ''
}

function fromInputValue(value: string): Date | undefined {
  if (!value) return undefined
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? undefined : date
}

export function DateTimeRangePicker(props: DateTimeRangePickerProps) {
  const { t, i18n } = useTranslation()
  const fieldId = useId()
  const [open, setOpen] = useState(false)
  const [draftStart, setDraftStart] = useState(toInputValue(props.start))
  const [draftEnd, setDraftEnd] = useState(toInputValue(props.end))
  const draftInvalid = Boolean(draftStart && draftEnd && draftStart > draftEnd)

  const label = useMemo(() => {
    if (!props.start && !props.end) return t('Date Range')
    const startText = props.start
      ? dayjs(props.start).format('YYYY-MM-DD HH:mm')
      : '-'
    const endText = props.end
      ? dayjs(props.end).format('YYYY-MM-DD HH:mm')
      : '-'
    return `${startText} ~ ${endText}`
  }, [props.end, props.start, t])

  const monthOptions = useMemo(() => {
    if (!props.monthOptionsCount) return []
    const formatter = new Intl.DateTimeFormat(
      toIntlLocale(i18n.resolvedLanguage || i18n.language),
      { year: 'numeric', month: 'long' }
    )
    const currentMonth = dayjs().startOf('month')
    return Array.from({ length: props.monthOptionsCount }, (_, index) => {
      const month = currentMonth.subtract(index, 'month')
      return {
        value: month.format('YYYY-MM'),
        label: formatter.format(month.toDate()),
      }
    })
  }, [i18n.language, i18n.resolvedLanguage, props.monthOptionsCount])

  const selectedMonth = useMemo(() => {
    if (!props.start || !props.end) return null
    const start = dayjs(props.start)
    const end = dayjs(props.end)
    const isFullMonth =
      start.isSame(end, 'month') &&
      start.isSame(start.startOf('month'), 'second') &&
      end.isSame(end.endOf('month'), 'second')
    return isFullMonth ? start.format('YYYY-MM') : null
  }, [props.end, props.start])

  const handleOpenChange = (nextOpen: boolean) => {
    if (nextOpen) {
      setDraftStart(toInputValue(props.start))
      setDraftEnd(toInputValue(props.end))
    }
    setOpen(nextOpen)
  }

  const applyDraft = () => {
    if (draftInvalid) return
    props.onChange({
      start: fromInputValue(draftStart),
      end: fromInputValue(draftEnd),
    })
    setOpen(false)
  }

  const applyPreset = (kind: 'today' | '7d' | 'week' | '30d' | 'month') => {
    const now = dayjs()
    const presets = {
      today: {
        start: now.startOf('day').toDate(),
        end: now.endOf('day').toDate(),
      },
      '7d': {
        start: now.subtract(6, 'day').startOf('day').toDate(),
        end: now.endOf('day').toDate(),
      },
      week: {
        start: now.startOf('week').toDate(),
        end: now.endOf('week').toDate(),
      },
      '30d': {
        start: now.subtract(29, 'day').startOf('day').toDate(),
        end: now.endOf('day').toDate(),
      },
      month: {
        start: now.startOf('month').toDate(),
        end: now.endOf('month').toDate(),
      },
    }
    const range = presets[kind]
    setDraftStart(toInputValue(range.start))
    setDraftEnd(toInputValue(range.end))
    props.onChange(range)
    setOpen(false)
  }

  const applyMonth = (value: string | null) => {
    if (!value) return
    const month = dayjs(`${value}-01`)
    const range = {
      start: month.startOf('month').toDate(),
      end: month.endOf('month').toDate(),
    }
    setDraftStart(toInputValue(range.start))
    setDraftEnd(toInputValue(range.end))
    props.onChange(range)
    setOpen(false)
  }

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger
        render={
          <Button
            type='button'
            variant='outline'
            className={cn(
              'w-full justify-start gap-2 px-2.5 text-sm leading-5 font-normal tabular-nums',
              !props.start && !props.end && 'text-muted-foreground',
              props.className
            )}
          />
        }
      >
        <CalendarDays
          data-icon='inline-start'
          className='text-muted-foreground shrink-0'
        />
        <span className='truncate'>{label}</span>
      </PopoverTrigger>
      <PopoverContent
        align='start'
        className='w-[min(520px,calc(100vw-2rem))] p-3'
      >
        <div className='space-y-3'>
          <div className='grid gap-2 sm:grid-cols-[1fr_auto_1fr] sm:items-end'>
            <div className='space-y-1.5'>
              <label
                htmlFor={`${fieldId}-start`}
                className='text-muted-foreground text-xs'
              >
                {t('Start Time')}
              </label>
              <Input
                id={`${fieldId}-start`}
                type='datetime-local'
                value={draftStart}
                max={draftEnd || undefined}
                aria-invalid={draftInvalid}
                onChange={(event) => setDraftStart(event.target.value)}
                className='h-8 text-sm leading-5 tabular-nums'
              />
            </div>
            <span className='text-muted-foreground hidden pb-2 text-xs sm:block'>
              ~
            </span>
            <div className='space-y-1.5'>
              <label
                htmlFor={`${fieldId}-end`}
                className='text-muted-foreground text-xs'
              >
                {t('End Time')}
              </label>
              <Input
                id={`${fieldId}-end`}
                type='datetime-local'
                value={draftEnd}
                min={draftStart || undefined}
                aria-invalid={draftInvalid}
                onChange={(event) => setDraftEnd(event.target.value)}
                className='h-8 text-sm leading-5 tabular-nums'
              />
            </div>
          </div>

          <div className='flex flex-wrap gap-1.5'>
            <Button
              type='button'
              variant='secondary'
              size='sm'
              className='h-7 flex-1 px-2 text-xs'
              onClick={() => applyPreset('today')}
            >
              {t('Today')}
            </Button>
            <Button
              type='button'
              variant='secondary'
              size='sm'
              className='h-7 flex-1 px-2 text-xs'
              onClick={() => applyPreset('7d')}
            >
              {t('7 Days')}
            </Button>
            <Button
              type='button'
              variant='secondary'
              size='sm'
              className='h-7 flex-1 px-2 text-xs'
              onClick={() => applyPreset('week')}
            >
              {t('This week')}
            </Button>
            <Button
              type='button'
              variant='secondary'
              size='sm'
              className='h-7 flex-1 px-2 text-xs'
              onClick={() => applyPreset('30d')}
            >
              {t('30 Days')}
            </Button>
            {props.monthOptionsCount ? (
              <Select
                items={monthOptions}
                value={selectedMonth}
                onValueChange={applyMonth}
              >
                <SelectTrigger size='sm' className='min-w-36 flex-1 text-xs'>
                  <SelectValue placeholder={t('Select month')} />
                </SelectTrigger>
                <SelectContent alignItemWithTrigger={false} align='end'>
                  <SelectGroup>
                    {monthOptions.map((month) => (
                      <SelectItem key={month.value} value={month.value}>
                        {month.label}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            ) : (
              <Button
                type='button'
                variant='secondary'
                size='sm'
                className='h-7 flex-1 px-2 text-xs'
                onClick={() => applyPreset('month')}
              >
                {t('This month')}
              </Button>
            )}
          </div>

          <div className='flex justify-end'>
            <Button
              size='sm'
              className='h-8'
              disabled={draftInvalid}
              onClick={applyDraft}
            >
              {t('Confirm')}
            </Button>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  )
}
