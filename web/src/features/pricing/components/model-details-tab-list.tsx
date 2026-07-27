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
import { Code2, HeartPulse, Info } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { TabsList, TabsTrigger } from '@/components/ui/tabs'
import { cn } from '@/lib/utils'

const TAB_VALUES = ['overview', 'performance', 'api'] as const
const BASIC_TAB_VALUES = ['overview', 'api'] as const
type TabValue = (typeof TAB_VALUES)[number]

const TAB_META: Record<
  TabValue,
  { icon: React.ComponentType<{ className?: string }>; labelKey: string }
> = {
  overview: { icon: Info, labelKey: 'Overview' },
  performance: { icon: HeartPulse, labelKey: 'Performance' },
  api: { icon: Code2, labelKey: 'API' },
}

export function ModelDetailsTabList(props: { showPerformance: boolean }) {
  const { t } = useTranslation()
  const visibleTabValues = props.showPerformance ? TAB_VALUES : BASIC_TAB_VALUES

  return (
    <TabsList
      className={cn(
        'bg-muted/60 grid w-full gap-1 rounded-lg p-1 group-data-horizontal/tabs:h-auto',
        props.showPerformance ? 'grid-cols-3' : 'grid-cols-2'
      )}
    >
      {visibleTabValues.map((value) => {
        const Icon = TAB_META[value].icon
        return (
          <TabsTrigger
            key={value}
            value={value}
            className='h-8 min-w-0 gap-1.5 rounded-md px-3 text-xs sm:text-sm'
          >
            <Icon className='size-3.5' />
            <span className='truncate'>{t(TAB_META[value].labelKey)}</span>
          </TabsTrigger>
        )
      })}
    </TabsList>
  )
}
