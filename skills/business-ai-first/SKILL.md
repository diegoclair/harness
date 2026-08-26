---
name: business-ai-first
description: Playbook de empresa AI-first (do 0 aos US$ 100M ARR) — a IA executa o trabalho e entrega resultado, o humano só confirma; mensalidade é cota de trabalho; margem é decisão de produto; o founder constrói o sistema de decisão. Use SEMPRE que o Diego desenhar, precificar, documentar, revisar ou implementar feature, plano, landing, copy, onboarding ou processo de qualquer produto dele — mesmo sem dizer "AI-first". Use também quando ele trouxer uma hipótese de negócio ("ouvi dizer que X"), pedir "como estamos?", quiser auditar playbooks, decidir contratação ou capital, ou quando uma feature soar como "permitir que o usuário faça X". Roda o ritual semanal (estágio → gargalo → hipótese → desconfirmar → repetir).
---

# business-ai-first

> *"O novo playbook não é fazer mais rápido aquilo que fazíamos antes. É aprender quais regras
> deixaram de valer."* — Edson Rigonatti (Astella), ago/2026

A skill é um **thinking partner e um ritual**, não uma lista: o valor está em aprender mais rápido
que o concorrente e transformar o aprendizado em regra escrita, tarefa agentificada, cliente
entregue. O conteúdo completo (mapa 0→100M, cinco playbooks old→AI-first, economia, moat, FDE,
founder) está em [references/playbook-ai-first.md](references/playbook-ai-first.md) — leia a seção
relevante pelo sumário quando precisar do argumento inteiro, não o arquivo todo.

## Postura: desconfirmar antes de documentar

Quando o Diego traz uma hipótese ("ouvi dizer que X", "acho que Y"), o trabalho é **tentar
derrubá-la** antes de registrar: buscar a fonte contrária, apontar o que o repo já diz que
contradiz, carimbar o que sobra pela régua de dados do produto (`oficial` / `estudo` / `opinião` /
`[hipótese]`). Concordar sem tentar é falha de função — IA acelera tudo, inclusive erro, e uma
hipótese não testada vira feature com custo de inferência.

## Três princípios que não mudam

1. **Fundamentos > velocidade.** Mapear tarefa e workflow (quem decide, com que informação, onde
   erra) **antes** de decidir o que vai pra LLM — e nem tudo vai: regra determinística, validação
   e cálculo ficam em código.
2. **Aprendizado contínuo.** Regra escrita com convicção **e** gatilho de revisão; derrubada por
   dado, muda no mesmo dia. Regra sem gatilho é dogma.
3. **Jornada é hábito.** Aprender, vender, contratar, levantar capital são hábitos com cadência.
   O que não tem cadência não acontece.

O founder é demitido do **output repetido**, não do contato com o detalhe. *Founder Mode* (PG,
set/24) e "herói → arquiteto" não se contradizem se a regra for a de Garry Tan (ago/26): **never
do one-off work** — mão na massa quando precisa, mas o que ele fez vira regra, template ou agente
no mesmo dia. Sinal de alerta: founder que só vê dashboard — o sistema está aprendendo sem ele, e
é assim que "hire good people and give them room" quebrou empresas.

## A frase e o teste

**AI-first é entrega de resultado.** Não se vende ferramenta pro cliente operar; vende-se o
trabalho feito pela IA, que o cliente confirma ou ignora. Toda feature passa por três perguntas:

1. **O que o cliente recebe pronto, sem operar nada?** "Uma tela onde ele pode fazer X" volta.
   Reescrita obrigatória: **IA sugere → cliente aceita → IA executa.**
   *Exemplo que fixou a regra:* "permitir criar mais de um anúncio por produto" (ferramenta) →
   "IA olha o catálogo, sugere os anúncios que valem existir com o porquê, seller aceita no
   Telegram, IA cria" (entrega).
2. **Qual a cota mensal de trabalho da IA que o plano inclui?** Mensalidade é cota de trabalho
   executado com excedente publicado — não acesso a botões, não por assento.
3. **Onde ainda há julgamento humano, e por quê?** Portão humano só onde plataforma, risco ou
   lei exigem. Onde é escolha nossa, encolhe conforme a taxa de aceite sobe.

Economia em uma linha: inferência e humano no portão são **COGS**; cada feature declara custo por
tarefa (medido), % humano e quem paga quanto por qual resultado. **Roteamento de modelo por
tarefa é a primeira alavanca de margem** (Cursor saiu de −23% pra positivo roteando metade do
tráfego pra modelo próprio/menor) — tarefa determinística em código, tarefa simples em modelo
pequeno, modelo grande só onde o erro custa; e margem de self-serve/SMB medida **separada** de
enterprise, porque costumam ter sinal oposto. O comparável de preço é **o custo de executar o
trabalho hoje** (hora humana), não o software concorrente. Moat = ponto de controle
(contexto · workflow · distribuição), nunca o modelo. Argumento completo na referência.

## O que a regra NÃO autoriza

- **Prometer resultado sem medição.** Ganho de conversão/ranking/vendas é `[hipótese]` até
  `medido` com controle. O que se vende no dia 1 é hora absorvida e erro de regra evitado.
- **Tirar o humano onde a plataforma exige** — registrar o porquê na ficha.
- **Serviço sem loop de FDE — virar software house.** Serviço humano é bem-vindo se roda
  *descobrir → executar → templatizar → produtizar* com a saída de cada passo escrita. O teste
  que separa service-as-software de software house é a **curva**: a entrega N tem que ser mais
  barata, mais rápida e mais lucrativa que a N−1, porque um padrão foi identificado e virou
  template ou agente. Entrega que custa o mesmo da anterior é hora vendida, não sistema. **É isso
  que o investidor AI-first avalia no founder: se ele sabe sistematizar trabalho humano.** Em
  produto de muitas contas pequenas (SMB), engenheiro embutido por cliente não fecha a conta
  (a16z, jan/26: FDE só compensa em poucas contas grandes) — o loop é **agente-first desde o
  descobrir**: o humano observa e corrige o agente, não faz no lugar dele. Limiar de produtização
  publicado: por volta da ~25ª integração custom (Decagon) / ~80 clientes (Crosby) as empresas
  passam a dizer não ao custom — se a sua ainda diz sim depois disso, virou serviço. Métrica pra
  acompanhar: **HURT — Human Review Time** por tarefa (Emergence), caindo.
- **Gastar antes de saber.** Capital compra velocidade, não conhecimento: entrevista, medição e
  leitura da doc vêm antes de mídia, time ou ferramenta.

## Ritual semanal (o procedimento)

Invocada num produto — ou quando o Diego pergunta "como estamos?" — a skill roda os cinco passos e
escreve uma entrada datada em `docs/product/ritual.md` (a anterior fica; é o histórico do sistema
aprendendo):

| # | Passo | Saída |
|---|---|---|
| 1 | **Localize o estágio** — 0→1, 1→10, 10→100? | estágio + evidência (Normandia respondida? pagantes? Ormuz?); em reais, a coluna do Astella Napkin (pre-seed R$ 0–1M · seed R$ 3,5–10M, 3x/ano · A R$ 18–30M · B R$ 50M+) e o que ela exige de produto, pessoas e máquina de vendas |
| 2 | **Nomeie o gargalo** — qual fundamento limita o throughput? | uma frase; e "a última coisa que automatizamos foi ele?" |
| 3 | **Escolha UMA hipótese** — qual regra antiga precisa ser revista? | a regra, o gatilho suspeito, o que a derrubaria |
| 4 | **Traga thinking partners** | a skill tenta derrubar (fonte, repo, número); o par humano registra discordância |
| 5 | **Repita** | diff nas regras; "não decidido, de propósito" também é saída; **ação da semana = o gargalo** |

Saída sem ação sobre o gargalo é ritual falhado; entrada que só confirma o que já se pensava é
sinal de que o passo 4 não rodou.

## Fichas que cada produto mantém

| Arquivo (em `docs/product/`) | O quê | Quando atualizar |
|---|---|---|
| `normandia.md` | as seis do PMF (quem compra · tarefa · outcome · % IA vs humano · custo de inferência · ponto de controle), prova em 5 dimensões, Ormuz (workflow / rail), FDE dos serviços, máquina de vendas, lista de demissões do founder | a cada rodada de cliente/entrevista |
| `playbooks.md` | auditoria dos cinco playbooks (produto · crescimento · operação · capital · founder): existe? passo de FDE? dono? entrega A→B? métrica? o que falta? | no ritual; playbook ausente é achado, não vergonha |
| `features-ai-first.md` | cada feature reescrita como sugere → aceita → executa, com cota, portão humano, eixo de moat e o que aprende com aceite/edição/recusa | a cada feature nova ou revista |
| `ritual.md` | uma entrada por semana | semanal |
| `measurements/` | custo por tarefa, taxa de aceite sem edição — o que só código responde | quando medir |

Essas fichas são o **company brain** do produto: Markdown versionado que agentes e pessoas leem
**antes** de escrever ou decidir. Cada ficha tem um dono nomeado e um gatilho de drift — no ritual,
a pergunta "o que aqui já não é verdade?" — porque regra que ninguém revisa envelhece em silêncio e
o agente segue obedecendo. Benchmark externo pro "% humano" das fichas: mediana ≥30% de tarefas com
intervenção humana em metade das empresas AI-native (ICONIQ, jul/26) — abaixo disso é raro, acima
é normal, mas a curva tem que cair.

Ferramenta ou entrega, em uma olhada: copy que diz "você pode" é ferramenta; "a IA faz" é entrega.

## Checklists

**Feature:** primeira linha diz o que a IA entrega pronto · reescrita sugere→aceita→executa ·
cota declarada · portão humano localizado e justificado · custo por tarefa medido ou "a medir"
com método · eixo de moat e o que aprende · promessa além de hora/erro carimbada `[hipótese]`.

**Preço:** unidade = trabalho executado · comparável = custo humano de fazer hoje · excedente
publicado, descer de faixa automático · margem por tarefa (contribution economics), não margem
SaaS do agregado · **a unidade tem que ser legível como resultado por cliente e investidor** —
"resposta enviada no prazo" é outcome, "mensagem processada" é uso com outro nome; se a cota
consome sem entregar (tarefa falhou, seller recusou), não conta. Híbrido base + resultado é o
padrão publicado (Bessemer 2026; puro outcome é minoria — ICONIQ jul/26: 23%); o que se mostra ao
investidor é a fatia da receita atrelada a resultado e a curva dela · **as quatro perguntas que
derrubam outcome pricing na prática** (Constellation, jun/26) respondidas por escrito antes de
publicar: o que é o outcome; como personaliza por cliente; quem audita o valor; quem carrega o
risco de custo. Sem elas, a cota vira briga de definição na renovação · **plano de uso em paralelo,
sempre** — o mercado outcome vende resolução mas a maioria fica em por-conversa/assento (Decagon,
Sierra, Harvey em assento; Lovable e Cursor migraram pra crédito/token por perder dinheiro em
heavy users); gatilho: se >70% dos clientes ficam no plano de uso, a definição de "resultado" está
cara demais de auditar — simplificar a unidade antes de insistir.

**Copy/landing:** cada bloco descreve entrega, não capacidade · nenhum número sem `medido` · **nenhum
compromisso público que dependa do que não se controla** (teto de reajuste, "publicamos todo mês",
SLA) — promessa quebrada custa mais pra quem prometeu do que a ausência custa pra quem nunca
prometeu; o número entra quando medido, com o método do lado, e a cadência não é promessa · a
tese da categoria com o nome do founder (autoridade), não o pitch da feature.

**Serviço / onboarding / entrega manual:** custo e horas desta entrega vs. a anterior (tem que
cair) · qual padrão esta entrega revelou · o que virou template ou agente por causa dela · o que
ainda pediu julgamento e por quê.

**Investidor / rodada:** em que coluna do Napkin a empresa está e o que falta pra próxima ·
V Multiple = pre-money / (lucro bruto × crescimento²) — margem AI-native de 50–60% cobra
crescimento ao quadrado, então mostrar a **curva** de margem (FDE) e não o ponto · efficiency
score (receita adicionada / burn) com inferência dentro do burn · fatia da receita atrelada a
resultado.

**Contratar / gastar:** o processo + agente absorve? (engineer around constraints) · o playbook
do cargo existe antes da vaga? · ataca o gargalo nomeado? · decisão de capital escrita com gatilho?

**Subagent que desenha ou implementa feature:** colar as três perguntas do teste e o checklist de
feature no prompt.

## Referências

- [references/playbook-ai-first.md](references/playbook-ai-first.md) — a masterclass inteira
  adaptada, com sumário: mapa 0→100M, Normandia, prova, Ormuz, sistematização, FDE, Growth
  Endurance, founder, os cinco playbooks old→AI-first, moat, economia, fontes.
- [references/service-as-software.md](references/service-as-software.md) — tese Sequoia /
  Foundation Capital: vender o trabalho, não o software. Leia ao precificar por resultado ou ao
  argumentar por que serviço ≠ resultado.
- [references/ai-first-2026.md](references/ai-first-2026.md) — o que mudou de 2024 a ago/2026:
  margens medidas (Bessemer, ICONIQ), outcome pricing em preço de lista e suas críticas, FDE como
  motion, company brain, MIT NANDA e os 5%, Brasil (Astella/Lastro). §7 diz o que confirma,
  contradiz ou acrescenta ao playbook — leia ao precificar, ao decidir serviço vs. agente, ou
  quando precisar de número externo.
- [references/astella-napkin.md](references/astella-napkin.md) — a régua do fundo do palestrante
  por estágio, em reais (receita, crescimento, rodada, pre-money, produto, pessoas, máquina de
  vendas, efficiency score) e o V Multiple. Leia ao localizar o estágio ou preparar conversa com
  investidor.
- [references/ai-first-yc-casos.md](references/ai-first-yc-casos.md) — YC 2025–26 (28% do W26 é
  "AI-native service"), Founder Mode vs. arquiteto, o que Sierra/Decagon/Harvey/Cursor/Lovable
  cobram de verdade e a margem do Cursor, limiares de produtização, HURT. Leia quando a regra
  parecer boa demais — é a referência das contradições.
- Caso vivo: `~/www/rednev/docs/product/` (normandia, playbooks, ritual, features-ai-first).
