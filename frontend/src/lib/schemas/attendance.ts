import { z } from 'zod';

// attendanceSchema is form-only: it carries `check_out_time` for the
// edit dialog, but the API splits create vs update DTOs differently
// (ChildAttendanceCreateRequest has only check_in_time + date;
// check_out_time is set via ChildAttendanceUpdateRequest). Form
// fields map to one of two API DTOs depending on flow, so no single
// satisfies target is appropriate.
export const attendanceSchema = z.object({
  status: z.enum(['present', 'absent', 'sick', 'vacation']),
  check_in_time: z.string().optional(),
  check_out_time: z.string().optional(),
  note: z.string().max(500).optional(),
});

export type AttendanceFormData = z.infer<typeof attendanceSchema>;
