import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { ComboboxInput } from '@/components/ui/combobox-input'
import { searchUsers } from '@/features/users/api'
import { useDebounce } from '@/hooks/use-debounce'

export function TransferUserFilter(props: {
  value: string
  onValueChange: (value: string) => void
}) {
  const { t } = useTranslation()
  const keyword = useDebounce(props.value.trim(), 300)
  const query = useQuery({
    queryKey: ['transfer-record-users', keyword],
    queryFn: async () => {
      const result = await searchUsers({ keyword, page_size: 20 })
      if (!result.success) throw new Error(result.message)
      return result.data?.items ?? []
    },
  })

  return (
    <ComboboxInput
      id='transfer-record-username'
      className='w-52'
      value={props.value}
      onValueChange={props.onValueChange}
      allowCustomValue
      placeholder={t('All users')}
      options={[
        { value: '', label: t('All users') },
        ...(query.data ?? []).map((user) => ({
          value: user.username,
          label: user.username,
        })),
      ]}
    />
  )
}
