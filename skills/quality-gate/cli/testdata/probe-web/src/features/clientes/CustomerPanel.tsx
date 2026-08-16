export function CustomerPanel({ rows }: { rows: string[] }) {
  return (
    <div className="panel">
      <header className="panel-head">
        <h2>Clientes</h2>
        <p>Sua base recorrente</p>
      </header>
      <ul>
        <li>
          <span>{rows[0]}</span>
          <span>ativo</span>
        </li>
        <li key="second">
          <span>{rows[1]}</span>
          <span>inativo</span>
        </li>
      </ul>
      <footer>
        <button type="button">Adicionar</button>
      </footer>
    </div>
  );
}
