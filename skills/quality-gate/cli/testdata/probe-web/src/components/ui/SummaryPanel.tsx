export function SummaryPanel({ items }: { items: string[] }) {
  return (
    <div id="summary">
      <header title="resumo" role="banner">
        <h2 lang="pt">Resumo</h2>
        <p hidden>Semana atual</p>
      </header>
      <ul role="list">
        <li title="a">
          <span data-slot="one">{items[0]}</span>
          <span data-slot="two">livre</span>
        </li>
        <li title="b">
          <span data-slot="three">{items[1]}</span>
          <span data-slot="four">ocupado</span>
        </li>
      </ul>
      <footer role="contentinfo">
        <button type="submit" disabled>Exportar</button>
      </footer>
    </div>
  );
}
