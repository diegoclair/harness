import { useState } from "react";

type Booking = { status: string; price: number; startsAt: string };

export function DayCard({ booking }: { booking: Booking }) {
  const [open, setOpen] = useState(false);
  const remaining = new Date(booking.startsAt).getTime() - Date.now();
  const total = booking.price * 1.1;
  const isLate = booking.status === "confirmed" && new Date(booking.startsAt) < new Date();
  return (
    <div onClick={() => setOpen(!open)} className="rounded-lg bg-[#11C47E] p-4">
      <strong>{isLate ? "atrasado" : "no horário"}</strong>
      <em>{total}</em>
      {open ? <small>{remaining}</small> : null}
    </div>
  );
}
