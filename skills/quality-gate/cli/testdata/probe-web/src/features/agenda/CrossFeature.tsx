import type { Customer } from "@/features/clientes/types";

export function CrossFeature({ customer }: { customer: Customer }) {
  return <label>{customer.name}</label>;
}
