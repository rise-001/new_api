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
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
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
  CUSTOM_MENU_LOCATION,
  CUSTOM_MENU_OPEN_MODE,
  isHttpUrl,
  type CustomMenuItem,
  type CustomMenuLocation,
  type CustomMenuOpenMode,
} from '@/lib/custom-menus'

const CUSTOM_MENU_FORM_ID = 'custom-menu-form'

const createCustomMenuSchema = (t: (key: string) => string) =>
  z
    .object({
      name: z.string().trim().min(1, t('Name is required')).max(64),
      url: z
        .string()
        .trim()
        .min(1, t('URL is required'))
        .max(2048)
        .refine(
          isHttpUrl,
          t('Provide a valid URL starting with http:// or https://')
        ),
      location: z.enum([
        CUSTOM_MENU_LOCATION.CHAT,
        CUSTOM_MENU_LOCATION.PERSONAL,
        CUSTOM_MENU_LOCATION.TOP,
      ]),
      openMode: z.enum([
        CUSTOM_MENU_OPEN_MODE.EMBED,
        CUSTOM_MENU_OPEN_MODE.NEW_TAB,
      ]),
    })
    .superRefine((value, context) => {
      if (
        value.location === CUSTOM_MENU_LOCATION.TOP &&
        value.openMode !== CUSTOM_MENU_OPEN_MODE.NEW_TAB
      ) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['openMode'],
          message: t('Top navigation links must open in a new tab'),
        })
      }
    })

export type CustomMenuFormValues = z.infer<
  ReturnType<typeof createCustomMenuSchema>
>

type CustomMenuDialogProps = {
  item: CustomMenuItem | null
  onOpenChange: (open: boolean) => void
  onSubmit: (values: CustomMenuFormValues) => void
}

export function CustomMenuDialog(props: CustomMenuDialogProps) {
  const { t } = useTranslation()
  const schema = createCustomMenuSchema(t)
  const form = useForm<CustomMenuFormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: props.item?.name ?? '',
      url: props.item?.url ?? '',
      location: props.item?.location ?? CUSTOM_MENU_LOCATION.CHAT,
      openMode: props.item?.openMode ?? CUSTOM_MENU_OPEN_MODE.EMBED,
    },
  })

  const location = form.watch('location')
  const locationOptions: Array<{
    value: CustomMenuLocation
    label: string
  }> = [
    { value: CUSTOM_MENU_LOCATION.CHAT, label: t('Chat area') },
    { value: CUSTOM_MENU_LOCATION.PERSONAL, label: t('Personal area') },
    { value: CUSTOM_MENU_LOCATION.TOP, label: t('Top navigation') },
  ]
  const openModeOptions: Array<{
    value: CustomMenuOpenMode
    label: string
  }> = [
    { value: CUSTOM_MENU_OPEN_MODE.EMBED, label: t('Embedded') },
    { value: CUSTOM_MENU_OPEN_MODE.NEW_TAB, label: t('New tab') },
  ]

  return (
    <Dialog
      open
      onOpenChange={props.onOpenChange}
      title={props.item ? t('Edit custom menu') : t('Add custom menu')}
      contentHeight='auto'
      footer={
        <>
          <Button
            type='button'
            variant='outline'
            onClick={() => props.onOpenChange(false)}
          >
            {t('Cancel')}
          </Button>
          <Button type='submit' form={CUSTOM_MENU_FORM_ID}>
            {props.item ? t('Update') : t('Add')}
          </Button>
        </>
      }
    >
      <Form {...form}>
        <form
          id={CUSTOM_MENU_FORM_ID}
          className='grid gap-4 sm:grid-cols-2'
          onSubmit={form.handleSubmit(props.onSubmit)}
        >
          <FormField
            control={form.control}
            name='name'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Name')}</FormLabel>
                <FormControl>
                  <Input {...field} autoFocus />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='url'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('URL')}</FormLabel>
                <FormControl>
                  <Input
                    {...field}
                    type='url'
                    placeholder='https://example.com'
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='location'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Display area')}</FormLabel>
                <Select
                  items={locationOptions}
                  value={field.value}
                  onValueChange={(value) => {
                    if (!value) return
                    const nextLocation = value as CustomMenuLocation
                    field.onChange(nextLocation)
                    if (nextLocation === CUSTOM_MENU_LOCATION.TOP) {
                      form.setValue('openMode', CUSTOM_MENU_OPEN_MODE.NEW_TAB, {
                        shouldDirty: true,
                        shouldValidate: true,
                      })
                    }
                  }}
                >
                  <FormControl>
                    <SelectTrigger className='w-full'>
                      <SelectValue />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent alignItemWithTrigger={false}>
                    <SelectGroup>
                      {locationOptions.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='openMode'
            render={({ field }) => (
              <FormItem data-disabled={location === CUSTOM_MENU_LOCATION.TOP}>
                <FormLabel>{t('Open mode')}</FormLabel>
                <Select
                  items={openModeOptions}
                  value={field.value}
                  disabled={location === CUSTOM_MENU_LOCATION.TOP}
                  onValueChange={(value) => {
                    if (value) field.onChange(value as CustomMenuOpenMode)
                  }}
                >
                  <FormControl>
                    <SelectTrigger className='w-full'>
                      <SelectValue />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent alignItemWithTrigger={false}>
                    <SelectGroup>
                      {openModeOptions.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
                <FormMessage />
              </FormItem>
            )}
          />
        </form>
      </Form>
    </Dialog>
  )
}
