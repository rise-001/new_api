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
import { zodResolver } from '@hookform/resolvers/zod'
import { useQuery } from '@tanstack/react-query'
import { Controller, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { DateTimePicker } from '@/components/datetime-picker'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { ComboboxInput } from '@/components/ui/combobox-input'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from '@/components/ui/field'
import { Spinner } from '@/components/ui/spinner'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { getApiKeys } from '@/features/keys/api'

import {
  consumptionExportFormSchema,
  type ConsumptionExportFormValues,
} from '../lib/schema'
import type { CreateConsumptionExportInput } from '../types'

type QuickRange = '24h' | '7d' | '30d' | 'month' | 'last-month'

type CreateExportDialogProps = {
  open: boolean
  pending: boolean
  onOpenChange: (open: boolean) => void
  onCreate: (input: CreateConsumptionExportInput) => Promise<boolean>
}

function defaultFormValues(): ConsumptionExportFormValues {
  const endTime = new Date()
  const startTime = new Date(endTime.getTime() - 7 * 24 * 60 * 60 * 1000)
  return {
    startTime,
    endTime,
    tokenId: 0,
    groupByToken: false,
    dailySummary: false,
  }
}

export function CreateExportDialog(props: CreateExportDialogProps) {
  const { t } = useTranslation()
  const form = useForm<ConsumptionExportFormValues>({
    resolver: zodResolver(consumptionExportFormSchema),
    defaultValues: defaultFormValues(),
  })

  const tokensQuery = useQuery({
    queryKey: ['api-keys', 'consumption-export-options'],
    queryFn: async () => {
      const pageSize = 100
      const firstPage = await getApiKeys({ p: 1, size: pageSize })
      if (!firstPage.success || !firstPage.data) {
        throw new Error(firstPage.message || t('Failed to load'))
      }

      const pageCount = Math.ceil(firstPage.data.total / pageSize)
      const remainingPages = await Promise.all(
        Array.from({ length: Math.max(0, pageCount - 1) }, (_, index) =>
          getApiKeys({ p: index + 2, size: pageSize })
        )
      )
      const tokens = [...firstPage.data.items]
      for (const page of remainingPages) {
        if (!page.success || !page.data) {
          throw new Error(page.message || t('Failed to load'))
        }
        tokens.push(...page.data.items)
      }
      return tokens
    },
    enabled: props.open,
    staleTime: 60 * 1000,
  })
  const tokenItems = [
    { value: '0', label: t('All API keys') },
    ...(tokensQuery.data ?? []).map((token) => ({
      value: String(token.id),
      label: token.name,
    })),
  ]

  const handleOpenChange = (open: boolean) => {
    if (open) form.reset(defaultFormValues())
    props.onOpenChange(open)
  }

  const applyQuickRange = (range: QuickRange) => {
    const now = new Date()
    let start = new Date(now)
    let end = now
    switch (range) {
      case '24h':
        start = new Date(now.getTime() - 24 * 60 * 60 * 1000)
        break
      case '7d':
        start = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000)
        break
      case '30d':
        start = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000)
        break
      case 'month':
        start = new Date(now.getFullYear(), now.getMonth(), 1)
        break
      case 'last-month':
        start = new Date(now.getFullYear(), now.getMonth() - 1, 1)
        end = new Date(now.getFullYear(), now.getMonth(), 0, 23, 59, 59)
        break
    }
    form.setValue('startTime', start, { shouldValidate: true })
    form.setValue('endTime', end, { shouldValidate: true })
  }

  const handleSubmit = form.handleSubmit(async (values) => {
    const created = await props.onCreate({
      start_timestamp: Math.floor(values.startTime.getTime() / 1000),
      end_timestamp: Math.floor(values.endTime.getTime() / 1000),
      token_id: values.tokenId,
      group_by_token: values.groupByToken,
      daily_summary: values.dailySummary,
      timezone_offset: new Date().getTimezoneOffset(),
    })
    if (created) props.onOpenChange(false)
  })

  return (
    <Dialog open={props.open} onOpenChange={handleOpenChange}>
      <DialogContent className='max-h-[calc(100vh-2rem)] overflow-y-auto sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>{t('Create consumption export')}</DialogTitle>
          <DialogDescription>
            {t('Choose the records and workbook layout for this export.')}
          </DialogDescription>
        </DialogHeader>

        <form id='consumption-export-form' onSubmit={handleSubmit}>
          <FieldGroup>
            <Field>
              <FieldLegend variant='label'>{t('Quick range')}</FieldLegend>
              <ToggleGroup
                spacing={2}
                variant='outline'
                className='flex w-full flex-wrap justify-start'
                onValueChange={(values) => {
                  const range = values[0] as QuickRange | undefined
                  if (range) applyQuickRange(range)
                }}
              >
                <ToggleGroupItem value='24h'>
                  {t('Last 24 hours')}
                </ToggleGroupItem>
                <ToggleGroupItem value='7d'>{t('Last 7 days')}</ToggleGroupItem>
                <ToggleGroupItem value='30d'>
                  {t('Last 30 days')}
                </ToggleGroupItem>
                <ToggleGroupItem value='month'>
                  {t('This month')}
                </ToggleGroupItem>
                <ToggleGroupItem value='last-month'>
                  {t('Last month')}
                </ToggleGroupItem>
              </ToggleGroup>
            </Field>

            <Controller
              control={form.control}
              name='startTime'
              render={({ field, fieldState }) => (
                <Field data-invalid={fieldState.invalid}>
                  <FieldLabel>{t('Start Time')}</FieldLabel>
                  <DateTimePicker
                    value={field.value}
                    onChange={(value) => {
                      if (value) field.onChange(value)
                    }}
                    placeholder={t('Select start time')}
                  />
                  <FieldError
                    errors={
                      fieldState.error?.message
                        ? [{ message: t(fieldState.error.message) }]
                        : undefined
                    }
                  />
                </Field>
              )}
            />

            <Controller
              control={form.control}
              name='endTime'
              render={({ field, fieldState }) => (
                <Field data-invalid={fieldState.invalid}>
                  <FieldLabel>{t('End Time')}</FieldLabel>
                  <DateTimePicker
                    value={field.value}
                    onChange={(value) => {
                      if (value) field.onChange(value)
                    }}
                    placeholder={t('Select end time')}
                  />
                  <FieldError
                    errors={
                      fieldState.error?.message
                        ? [{ message: t(fieldState.error.message) }]
                        : undefined
                    }
                  />
                </Field>
              )}
            />

            <Controller
              control={form.control}
              name='tokenId'
              render={({ field }) => (
                <Field data-invalid={tokensQuery.isError}>
                  <FieldLabel htmlFor='consumption-export-token'>
                    {t('API key')}
                  </FieldLabel>
                  <ComboboxInput
                    id='consumption-export-token'
                    options={tokenItems}
                    value={String(field.value)}
                    onValueChange={(value) => field.onChange(Number(value))}
                    placeholder={t('All API keys')}
                    emptyText='No data'
                    className='w-full'
                  />
                  <FieldDescription>
                    {t(
                      'Select all API keys to include every consumption record.'
                    )}
                  </FieldDescription>
                  <FieldError
                    errors={
                      tokensQuery.isError
                        ? [{ message: t('Failed to load') }]
                        : undefined
                    }
                  />
                </Field>
              )}
            />

            <FieldSet>
              <FieldLegend variant='label'>{t('Workbook options')}</FieldLegend>
              <FieldGroup className='gap-3'>
                <Controller
                  control={form.control}
                  name='groupByToken'
                  render={({ field }) => (
                    <Field orientation='horizontal'>
                      <Checkbox
                        id='group-by-token'
                        checked={field.value}
                        onCheckedChange={(checked) => {
                          field.onChange(checked === true)
                          if (checked === true) {
                            form.setValue('dailySummary', false)
                          }
                        }}
                      />
                      <FieldContent>
                        <FieldLabel
                          htmlFor='group-by-token'
                          className='font-normal'
                        >
                          {t('Create one sheet per API key')}
                        </FieldLabel>
                      </FieldContent>
                    </Field>
                  )}
                />
                <Controller
                  control={form.control}
                  name='dailySummary'
                  render={({ field }) => (
                    <Field orientation='horizontal'>
                      <Checkbox
                        id='daily-summary'
                        checked={field.value}
                        onCheckedChange={(checked) => {
                          field.onChange(checked === true)
                          if (checked === true) {
                            form.setValue('groupByToken', false)
                          }
                        }}
                      />
                      <FieldContent>
                        <FieldLabel
                          htmlFor='daily-summary'
                          className='font-normal'
                        >
                          {t('Export daily summaries without detail rows')}
                        </FieldLabel>
                      </FieldContent>
                    </Field>
                  )}
                />
              </FieldGroup>
            </FieldSet>
          </FieldGroup>
        </form>

        <DialogFooter>
          <Button
            type='button'
            variant='outline'
            onClick={() => props.onOpenChange(false)}
          >
            {t('Cancel')}
          </Button>
          <Button
            type='submit'
            form='consumption-export-form'
            disabled={props.pending}
          >
            {props.pending && <Spinner data-icon='inline-start' />}
            {t('Download')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
