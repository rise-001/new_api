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
import { z } from 'zod'

export const consumptionExportFormSchema = z
  .object({
    startTime: z.date(),
    endTime: z.date(),
    tokenId: z.number().int().min(0),
    groupByToken: z.boolean(),
    dailySummary: z.boolean(),
  })
  .superRefine((value, context) => {
    if (value.endTime.getTime() <= value.startTime.getTime()) {
      context.addIssue({
        code: 'custom',
        path: ['endTime'],
        message: 'End time must be later than start time',
      })
      return
    }
    const maxRange = 366 * 24 * 60 * 60 * 1000
    if (value.endTime.getTime() - value.startTime.getTime() > maxRange) {
      context.addIssue({
        code: 'custom',
        path: ['endTime'],
        message: 'Export time range cannot exceed 366 days',
      })
    }
  })

export type ConsumptionExportFormValues = z.infer<
  typeof consumptionExportFormSchema
>
