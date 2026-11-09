import { z } from "zod";

// The venue picker on a project: which directory venue, and the price snapshot
// agreed for this project (0 = not set, same convention as the venue form).
export const projectVenueSchema = z.object({
  venueId: z.string().min(1, "Pilih venue dari daftar"),
  rentalPrice: z.coerce.number().min(0, "Harga sewa tidak boleh negatif"),
  charge: z.coerce.number().min(0, "Charge tidak boleh negatif"),
});

export type ProjectVenueFormValues = z.infer<typeof projectVenueSchema>;
