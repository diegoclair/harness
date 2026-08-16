export const agendaService = {
  async list(day: string): Promise<string> {
    const response = await fetch(`/provider/agenda/${day}/`);
    if (!response.ok) {
      throw new Error("agenda");
    }
    return response.text();
  },
};
