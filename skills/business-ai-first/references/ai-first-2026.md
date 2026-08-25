# AI-first como modelo operacional — atualização ago/2026

**Data:** 24/ago/2026. Continuação de `service-as-software.md` (material 2024: Sequoia "Act o1", Foundation Capital).
**Lido (página inteira):** Sequoia "2026: This is AGI" (jan/26), Sequoia/Bek "Services: The New Software" (mar/26), Fortune sobre Bek (abr/26), cobertura AI Ascent 2026 (theaiopportunities, mai/26), Sequoia podcast Bret Taylor, Bessemer "AI pricing playbook" (fev/26), Bessemer "Owning the outcome" (jul/26), Bessemer State of AI 2025 (PDF), ICONIQ State of AI jul/26 (site + SaaSletter + Upstarts jan/26), Menlo State of GenAI 2025 (dez/25), Foundation Capital "Lessons from year one" e "When model providers eat everything", a16z "Palantirization of everything" (jan/26), Pragmatic Engineer FDE (ago/25), Fortune/Yahoo e agentmodeai/chokmah sobre MIT NANDA, Constellation e Pricing Conundrum sobre outcome pricing, Tanay Jaipuria margens (set/25), YC RFS 2026 via Forbes, Anthropic "Founder's Playbook" (mai/26), the-ai-corner OS one-person (jun/26), SandsDX company brain (mar–jul/26), Superkind company brain (mai/26), 36kr sobre a16z (jan/25), Forbes one-person (abr/26), Astella/Lastro (ago/26), startups.com.br Rigonatti (mar/26).
**Não lido (403/402/503/404):** Forbes MIT 26/ago/25 (403); PDF primário MIT NANDA em mlq.ai (403); artigo Box "The Company Brain" no X (402); SaaStr 10 métricas ICONIQ (403); secondorderlabs margens (503); página de Rigonatti na Astella (404); Pragmatic Engineer paywall parcial (seção Ramp).

## §1 O que mudou desde 2024 na tese

- **De "act o1" para "ano dos agentes"**: Sequoia declara que o terceiro ingrediente (agente de longo horizonte) chegou em jan/26 e que "the AI applications of 2026 and 2027 will be doers. They will feel like colleagues" (Grady/Huang, jan/26, `opinião`). Huang no AI Ascent (mai/26): "Services, not software, becomes the addressable opportunity"; espectro de maturidade autocomplete → agentic → async → "dark factories" sem revisão humana (`opinião`).
- **A tese ganhou nome e tamanho**: Bek/Sequoia (mar/26): "A copilot sells the tool. An autopilot sells the work"; "if you sell the tool, you're in a race against the model"; US$ 6 em serviços para cada US$ 1 em software (`opinião`, número sem fonte no texto). AI Ascent fala em ~US$ 10 tri de serviços endereçáveis (`opinião`).
- **Margem deixou de ser debate e virou número medido**: Bessemer fev/26 50–60% vs 80–90% SaaS; ICONIQ 45% (2025) → 53% (2026e) → 59% (2027e). Consenso: abaixo de SaaS, subindo (`estudo`).
- **Outcome pricing saiu da tese e entrou no preço de lista dos incumbentes**: Salesforce Help Agent US$ 2/resolução (jun/26), Intercom Fin US$ 0,99/resolução, HubSpot US$ 0,50/conversa resolvida, Zendesk ~US$ 1,50–2 (`oficial`, via imprensa). Mas só 23% das empresas de IA usam outcome-based (ICONIQ jul/26, `estudo`); a recomendação dominante é **híbrido** (base + outcome).
- **Bessemer reposiciona a categoria como "AI-native services"** (jul/26): a IA "reaches the delivery layer itself"; "every delivered unit of work doubles as a training signal" (`opinião`). Foundation Capital: "in enterprise AI, integration is not a post-sale activity. It is the product surface" (jul/25, `opinião`).
- **FDE virou função padrão e virou alerta**: vagas +800–1000% em 2025 (a16z, jan/26, `imprensa`); 50% das empresas de IA planejam FDE como motion permanente (ICONIQ jul/26, `estudo`); a16z: "most companies copying the aesthetic are setting themselves up to become expensive services businesses with a software valuation multiple" (`opinião`).
- **Falha é a regra e é organizacional, não de modelo**: MIT NANDA jul/25 — 95% dos pilotos sem impacto no P&L; o gap é de aprendizado/workflow (`estudo`, metodologia contestada). ICONIQ jul/26: quase metade dos agentes exige intervenção humana em ≥30% das tarefas; falha típica "multi-step workflows that break partway through" (`estudo`).
- **Founder como orquestrador virou playbook publicado por lab**: Anthropic "Founder's Playbook" (mai/26): "the founder's role is shifting from individual contributor to orchestrator"; estágio Launch tem um "operating system" que troca atenção do founder por workflows agênticos (`oficial` do fornecedor, `opinião` na substância).

## §2 Números novos

| Métrica | Valor | Fonte | Data | Carimbo |
|---|---|---|---|---|
| Margem bruta empresa de IA vs SaaS | 50–60% vs 80–90% | Bessemer, AI pricing playbook | fev/26 | estudo |
| Margem bruta IA (todas), série | 41% (2024) · 45% (2025) · 52–53% (2026e) · 59% (2027e) | ICONIQ, ~300 execs, dez/25 e Q2/26 | jan/26, jul/26 | estudo |
| Margem bruta só camada de aplicação | 33% (2024) · 38% (2025) · 45% (2026e) · 60% (2027e) | ICONIQ | jan/26, jul/26 | estudo |
| Supernova vs Shooting Star | ~25% (freq. negativa) vs 60% margem; US$ 1,133M vs US$ 164K ARR/FTE; US$ 125M ARR ano 2 vs US$ 103M ano 4 | Bessemer State of AI 2025 (PDF) | ago/25 | estudo |
| Margem de AI-native services (Sierra) | ~70% vs 90% SaaS puro | Bek citando Bret Taylor, Fortune | abr/26 | opinião |
| Margem de model providers | OpenAI ~50%, Anthropic ~60% | Tanay Jaipuria citando The Information | set/25 | imprensa |
| Pricing: consumo / outcome | 35%→42% / 18%→23% em 6 meses; média 1,7 modelos por empresa | ICONIQ | jul/26 | estudo |
| Outcome atrelado a quê | economia de custo 36%; receita gerada 18% | ICONIQ | jul/26 | estudo |
| Custo por query melhorou ≥10% | 66% das empresas; 19% melhoraram ≥30% | ICONIQ | jul/26 | estudo |
| Intervenção humana em agentes | ~metade das empresas: humano em ≥30% das tarefas; ganho médio de produtividade <30% | ICONIQ | jul/26 | estudo |
| FDE como motion permanente | 50% planejam; 38% usam como driver de receita | ICONIQ | jul/26 | estudo |
| Vagas FDE | +800–1000% em 2025 | a16z citando job postings | jan/26 | imprensa |
| Preço por resolução | Salesforce US$ 2 · Intercom US$ 0,99 · HubSpot US$ 0,50 · Zendesk US$ 1,50–2 · sem cobrança se escalar pra humano | Constellation, Bessemer, Pricing Conundrum | fev–jun/26 | oficial via imprensa |
| Gasto enterprise em IA | US$ 37 bi (3,2× YoY); 76% comprado vs 24% construído (era 47/53) | Menlo, ~500 decisores | dez/25 | estudo |
| Conversão de venda IA vs SaaS | 47% vs 25% | Menlo | dez/25 | estudo |
| Deploy real de agentes | 16% enterprise / 27% startups | Menlo | dez/25 | estudo |
| Pilotos sem impacto no P&L | 95% (300 deploys, 150 entrevistas, 350 surveys); vendor 67% vs interno ~33% de sucesso | MIT NANDA | jul/25 | estudo (não peer-reviewed) |
| Serviços profissionais nos EUA | ~13% do PIB, 10× o software | Bessemer | jul/26 | opinião |
| Preço por milhão de tokens | Claude Sonnet 4.6 US$ 3/15 · Opus 4.6 US$ 5/25 · Gemini 3.1 Flash US$ 0,10/0,40 · Llama 4 Maverick US$ 0,22–0,27 in | Featherless; agregadores | mar/26 | imprensa |
| Queda de custo de inferência | 30–50%/ano open-source; "equivalente a GPT-4" US$ 20 → US$ 0,40 em 3 anos | Featherless; GPUnex | 2026 | imprensa, sem fonte primária |
| Preço de modelo de ponta | "stayed steady or even gone up" enquanto inferência cai 80–90%/ano | Tanay Jaipuria | set/25 | opinião |
| Stack agêntico de um founder solo | US$ 300–500/mês substitui US$ 80–120K/ano de payroll | cobertura do Founder's Playbook | mai/26 | imprensa |
| Seed AI sem receita | 40%+ | Forbes sobre YC RFS 2026 | fev/26 | imprensa |

Não apurado: número consolidado de "% de contratos outcome-based" em base de contratos (só há % de empresas que usam o modelo); preço por token oficial de ago/26 (fontes lidas são de mar/26).

## §3 Modelo operacional AI-first / company brain

**Consenso (3+ autores independentes):**
- Founder vira orquestrador; a empresa se organiza em torno de agentes que executam e humanos que julgam. Sequoia: "you'll go from working as an IC to managing a team of agents" (jan/26). Anthropic Founder's Playbook (mai/26). the-ai-corner (jun/26): "your edge moves from execution to judgment". YC RFS 2026: 6 de 7 apostas são "automate work itself", incl. "AI-native agencies (scaled services without staff growth)".
- Contexto proprietário é o ativo. Bessemer 2025: "memory and context are the new moats"; "systems of action are replacing systems of record". Foundation Capital: "the highest leverage asset isn't a legacy dataset: it's a feedback loop with real users in a real workflow". Bessemer jul/26: "the firm gets its environment for free".
- Redesenhar workflow, não acoplar chat. Grady (AI Ascent): "faster horses" (10–40%) vs "cars" (10–40×) exige reestruturar o workflow. MIT: os 5% redesenham o processo em torno da IA, não retrofitam ferramenta.

**Company brain / playbook como código (um a dois autores, não consenso):**
- SandsDX (mar/26): "a structured knowledge base that every AI agent reads before it writes: Markdown files in a git repository covering your ICP, personas, use cases, messaging framework, brand voice, and governance rules". Quatro propriedades: shared, enforceable (lint no git), evolving (PR + histórico do porquê), agent-readable (frontmatter, headings, citação `[Source: icp.md#...]`). Governança: um dono; outros times abrem issue, não editam; checagem mensal de drift. Inclui anti-ICP e linguagem proibida (`opinião`).
- Superkind (mai/26) usa o mesmo termo para camada RAG sobre wiki/CRM/e-mail, consultável por humano e agente — definição mais frouxa, sem "regras". Box publicou "The Company Brain" (X, 2026) — não lido (402).
- the-ai-corner (jun/26): OS do founder solo = framework de decisão (chat/cowork/code) + 4 estágios com critérios de saída + `CLAUDE.md` como sistema de contexto + catálogo de failure modes com prompts corretivos (`opinião`).
- [interpretação] "Company brain" em 2026 tem dois sentidos que não se falam: (a) repositório versionado de regras que agentes leem (SandsDX, Anthropic/CLAUDE.md) — é o "playbook como código"; (b) RAG corporativo. Para a skill interessa (a). Nenhum texto lido mede efeito; tudo é `opinião`.
- "One-person unicorn": Altman (2024) e Amodei ("2026", 70–80% de confiança, mai/25) previram; até abr/26 os casos citados pela Forbes são saídas de ~US$ 80M (Base44/Wix), aquisição OpenClaw, e US$ 110K/mês (Daymaker) — nenhum bilionário solo (`imprensa`).

## §4 FDE como modelo

- Definição operacional (Palantir via Pragmatic Engineer, ago/25): "one customer, many capabilities" vs. dev tradicional "one capability, many customers"; fases scoping → validation → delivery. OpenAI montou FDE no início de 2025 (Colin Jarvis): "often what the customer describes in scoping doesn't match the data/system reality" (`imprensa`).
- Risco nomeado pelos dois lados: Orosz — a linha entre consultor e engenheiro "can be blurred"; a16z (Andrusko, jan/26) — Palantir funciona porque serviço é "a means to drive product adoption, not the primary revenue stream"; teste: cliente maduro precisa de "meaningfully" menos embed com o tempo, senão "you're 'Accenture for X' with a nicer front-end". Pergunta-guia: "what is the minimum amount of Palantir-style forward deployment we need to bridge the AI adoption gap?" (`opinião`).
- Gate do a16z para usar FDE: problema mission-critical (não eficiência de 8–20%), dezenas de contas grandes (não milhares pequenas), workflows parecidos entre clientes, setor regulado (`opinião`).
- Foundation Capital (jul/25): pré e pós-venda se fundem — "the customer now expects to experience functionality, integration, and outcome before a contract is signed"; kit de ingestão cortou 70% do tempo de deploy num caso (`opinião`, número de portfólio).
- Dado: 50% planejam FDE permanente, 38% como driver de receita (ICONIQ jul/26, `estudo`). [interpretação] Isso é exatamente o sintoma que a16z alerta: FDE virando receita em vez de aprendizado.

## §5 Críticas e falhas

- **MIT NANDA (jul/25)**: 95% de 300 deploys sem impacto mensurável no P&L; 5% "integrated systems creating significant value". Os 5%: compram em vez de construir (67% vs ~33%), atacam back-office com fila (exceções de fatura, reconciliação), medem mudança de workflow (cycle time, erro, custo) e não adoção; ferramentas com "memory and learning loops". Budget: >50% em vendas/marketing, melhor ROI em back-office. Challapally: ferramentas genéricas "stall in enterprise use since they don't learn from or adapt to workflows" (`estudo`). Críticas: amostra não aleatória, 52 entrevistas na base qualitativa, horizonte ~6 meses, sem peer review, sem baseline pré-deploy em muitas empresas — "no measurable impact" pode ser falta de medição (`opinião`, agentmodeai).
- **Outcome pricing**: Constellation (jun/26) — quatro perguntas sem resposta: o que é o outcome, como personalizar por cliente, quem audita o valor, quem carrega o risco de custo; "haggling over definitions and fuzzy math". Pricing Conundrum (jun/26): "resolução" por inatividade de 72h não distingue satisfeito de "quiet defector"; lead "recomendado" sem conversão verificada; classificação favorece o vendor. Bessemer: falha em "soft ROI" (copilots de aconselhamento) e no "2026 renewal cliff". Bret Taylor: modelo "really disrupts the way you build a software company" — incumbentes têm "active disassociation with the outcomes"; procurement com budget fixo prefere assinatura (`opinião`). Bek admite: RFPs ainda pedem hora; GTM de serviços "remain unsolved"; regulação limita insourcing (auditoria).
- **Margem**: Supernovas frequentemente negativas (Bessemer); Anthropic "losing tens of thousands of dollars per month on a user on a $200 plan" (Jaipuria citando imprensa, set/25); ICONIQ: "too few fixed costs to generate software economics". Jaipuria: queda de inferência só ajuda quem roteia pra modelo barato; topo de linha não baixou (`opinião`).
- **Startups que viram serviço**: a16z (jan/26) é o texto direto — FDE copiado vira "expensive services business with a software valuation multiple". Dado quantitativo de quantas viraram: **não apurado**.
- **AI washing**: SEC colocou IA como prioridade de exame FY2026; onda criminal 2025–26 (CaaStle US$ 300M, SKAEL, IRL, Done Global, AllHere, ComplYant) por produtos "que exigiam grau significativo de envolvimento humano" apresentados como automação; 53 class actions ligadas a IA no 1S25 (Stanford SCAC) (`imprensa`/`oficial`). [interpretação] Declarar % humano por feature é defesa jurídica, não só honestidade de produto.
- **Model provider como concorrente**: Foundation Capital — "the model provider that powers you can also turn around and steamroll you"; labs replicam "every major system of record" como ambiente de treino; defesa "lives in the parts of the tech stack that big model providers won't bother to own" (`opinião`).

## §6 Brasil

- **Astella**: masterclass Rigonatti "AI-First: o novo playbook" (24/ago/26, assistida pelo Diego — fonte primária da skill). Publicado: entrevista startups.com.br (mar/26) — "o que muda é a camada, a casca, a forma de fazer, mas a base continua"; empresas passam a pagar por agente em vez de assinatura (`opinião`). Podcast Astella Playbook com Allan Paladino/Lastro (ago/26): refundação em torno de IA (Lais, WhatsApp, até +60% conversão lead→visita, número da empresa), "COGS é o novo CAC", moat = tecnologia + serviço + suporte em problema tão local que "big tech nem entende as palavras que os descrevem"; pipeline humano de venda/implantação/suporte mantido de propósito ("barriga no balcão"), sendo produtizado; 100% digital pra SMB brasileiro "impraticável no curto/médio prazo" (`opinião`). Site pessoal de Rigonatti sem posts; página na Astella 404.
- **Canary, Kaszek, monashees**: só notícia de deal (Kaszek/TravelX "AI-native", `imprensa`). Publicação de tese: **não apurado**.

## §7 Contra o playbook de Rigonatti (SKILL.md)

- **Confirma** — "AI-first é entrega de resultado": Bek "autopilot sells the work"; Bessemer "the outcome is the product"; YC "AI-native agencies". Consenso de 4+ VCs.
- **Confirma** — "margem é decisão de produto, não margem SaaS": Bessemer 50–60%, ICONIQ 45→59%, Supernovas negativas. Acrescenta o alvo: camada de aplicação madura projeta ~60% (ICONIQ 2027e); Sierra ~70%.
- **Confirma** — "comparável é o custo de executar o trabalho hoje": Bek 6:1 serviços/software; a16z clínica de US$ 500/ano de software para US$ 20K/ano de trabalho; Bessemer 13% do PIB.
- **Confirma** — "serviço sem loop de FDE = software house": a16z "Accenture for X"; teste de "meaningfully less embed" por cliente maduro é o mesmo teste de curva N < N−1 da skill.
- **Confirma** — "portão humano encolhe com taxa de aceite": Salesforce/Sierra cobram zero quando escala pra humano — o mercado já precifica o portão como custo do vendor.
- **Confirma** — moat = ponto de controle (contexto · workflow · distribuição): Bessemer "memory and context are the new moats"; Foundation "feedback loop in a real workflow"; Grady "customer intimacy, workflow integration, earned trust".
- **Contradiz (parcial)** — "mensalidade = cota de trabalho com excedente": os dados mostram que puro outcome é minoria (23%) e a recomendação de Bessemer e Taylor é **híbrido** (base + outcome), porque procurement quer previsibilidade e outcome gera briga de definição. A skill já é híbrida (cota + excedente), mas não trata as quatro perguntas de Constellation (definição, personalização, auditoria, risco de custo) — acrescentar ao checklist de preço.
- **Contradiz (parcial)** — FDE "descobrir → executar → templatizar → produtizar" pressupõe que serviço vale a pena; a16z diz que FDE só compensa em problema mission-critical com poucas contas grandes. Rednev é milhares de contas pequenas — o loop tem que ser agente-first desde a descoberta, não engenheiro embutido. [interpretação]
- **Acrescenta** — "company brain" como artefato: regras em Markdown versionadas, um dono, outros abrem issue, lint de consistência, checagem mensal de drift, anti-ICP e linguagem proibida (SandsDX). A skill tem fichas em `docs/product/`; falta declarar dono, gatilho de drift e que agentes leem antes de escrever.
- **Acrescenta** — métrica de intervenção: ICONIQ mede "% de tarefas com intervenção humana" (mediana ≥30% em metade das empresas) — é o benchmark externo pra "% humano" das fichas.
- **Acrescenta** — o que os 5% do MIT fazem é uma lista operacional: back-office com fila, medir cycle time/erro/custo antes e depois, comprar loop de aprendizado. Mapeia direto em `measurements/`.
- **Acrescenta** — risco jurídico de % humano não declarado (AI washing, SEC FY2026) — justificativa extra pra "não prometer resultado sem medição".
- **Acrescenta** — Foundation Capital: pré-venda = POC com dado real do cliente; "integration is the product surface". Onboarding é feature, não serviço.
- **Acrescenta** — Lastro (Astella): "COGS é o novo CAC"; complexidade brasileira como moat. Caso BR do playbook.

## §8 Fontes

| Fonte | URL | Data | Carimbo |
|---|---|---|---|
| Sequoia, Grady & Huang, "2026: This is AGI" | https://sequoiacap.com/article/2026-this-is-agi | 14/jan/26 | opinião |
| Sequoia, Bek, "Services: The New Software" | https://sequoiacap.com/article/services-the-new-software | 05/mar/26 | opinião |
| Fortune, Bek entrevista | https://fortune.com/2026/04/21/services-are-the-new-software-sequoia-venture-capital-julien-bek-ai-native-eye-on-ai/ | 21/abr/26 | imprensa |
| theaiopportunities, AI Ascent 2026 | https://www.theaiopportunities.com/p/sequoia-ai-ascent-2026-the-future | 15/mai/26 | imprensa |
| Sequoia podcast, Bret Taylor | https://sequoiacap.com/podcast/training-data-bret-taylor | 2025–26 (data não apurada) | opinião |
| Bessemer, AI pricing playbook | https://www.bvp.com/atlas/the-ai-pricing-and-monetization-playbook | 09/fev/26 | estudo |
| Bessemer, Owning the outcome | https://www.bvp.com/atlas/owning-the-outcome-bessemers-ai-native-services-evaluation-framework | 31/jul/26 | opinião |
| Bessemer, State of AI 2025 (PDF) | https://www.bvp.com/assets/uploads/2025/08/Final_PDF_State_of_AI_2025_slides_Bessemer_Venture_Partners.pdf | ago/25 | estudo |
| ICONIQ, State of AI 2026 | https://www.iconiq.com/growth/reports/state-of-ai-2026 | jul/26 | estudo |
| SaaSletter sobre ICONIQ | https://www.saasletter.com/p/iconiq-state-of-ai-july-2026-ai-revenue-mix | jul/26 | imprensa |
| Upstarts (Konrad) sobre ICONIQ | https://www.upstartsmedia.com/p/data-ai-startup-margins-rise | 28/jan/26 | imprensa |
| Menlo, State of GenAI 2025 | https://menlovc.com/perspective/2025-the-state-of-generative-ai-in-the-enterprise/ | 09/dez/25 | estudo |
| Foundation Capital, Lessons from year one | https://foundationcapital.com/the-4-6t-service-as-software-opportunity-lessons-from-year-one/ | jul/25 | opinião |
| Foundation Capital, When model providers eat everything | https://foundationcapital.com/ideas/when-model-providers-eat-everything-a-survival-guide-for-service-as-software-startups | 2025–26 (data não apurada) | opinião |
| a16z, Andrusko, Palantirization of everything | https://a16z.com/the-palantirization-of-everything/ | 16/jan/26 | opinião |
| 36kr sobre relatório a16z apps | https://eu.36kr.com/en/p/3647330224370436 | 20/jan/25 | imprensa |
| Pragmatic Engineer, FDE | https://newsletter.pragmaticengineer.com/p/forward-deployed-engineers | 12/ago/25 | imprensa |
| Fortune/Yahoo, MIT NANDA | https://finance.yahoo.com/news/mit-report-95-generative-ai-105412686.html | 18/ago/25 | imprensa |
| agentmodeai, crítica MIT | https://agentmodeai.com/the-mit-genai-pilot-failure-claim/ | 2025–26 | opinião |
| Chokmah, os 5% | https://chokmah.in/pov/why-95-percent-of-genai-pilots-fail/ | 2025 | opinião |
| MIT NANDA PDF (não lido, 403) | https://mlq.ai/media/quarterly_decks/v0.1_State_of_AI_in_Business_2025_Report.pdf | jul/25 | estudo |
| Forbes MIT (não lido, 403) | https://www.forbes.com/sites/jasonsnyder/2025/08/26/mit-finds-95-of-genai-pilots-fail-because-companies-avoid-friction/ | 26/ago/25 | imprensa |
| Constellation, Salesforce Help Agent | https://www.constellationr.com/insights/news/salesforce-takes-run-outcome-based-help-agent-pricing | 26/jun/26 | imprensa |
| Pricing Conundrum, Dholakia | https://thepricingconundrum.substack.com/p/outcome-based-pricing-in-practice | 09/jun/26 | opinião |
| Tanay Jaipuria, margens | https://www.tanayj.com/p/the-gross-margin-debate-in-ai | 02/set/25 | opinião |
| Featherless, preços por token | https://featherless.ai/blog/llm-api-pricing-comparison-2026-complete-guide-inference-costs | 04/mar/26 | imprensa |
| Forbes, YC RFS 2026 | https://www.forbes.com/sites/josipamajic/2026/02/04/ycs-2026-roadmap-signals-a-shift-from-human-augmented-to-ai-native-startups/ | 04/fev/26 | imprensa |
| Anthropic, Founder's Playbook | https://claude.com/blog/the-founders-playbook | 14/mai/26 | oficial (fornecedor) |
| the-ai-corner, OS one-person | https://www.the-ai-corner.com/p/one-person-startup-operating-system-2026 | 11/jun/26 | opinião |
| SandsDX, company brain | https://sandsdx.com/perspectives/executive/company-brain-for-ai-agents/ | 24/mar/26, upd. 25/jul/26 | opinião |
| Superkind, company brain | https://superkind.ai/ai-lexicon/company-brain | 21/mai/26 | opinião |
| Box, The Company Brain (não lido, 402) | https://x.com/Box/article/2064092437073867230 | 2026 | opinião |
| Forbes, one-person startups | https://www.forbes.com/sites/sandycarter/2026/04/04/openai-called-the-one-person-ai-startup-and-three-founders--proved-it/ | 04/abr/26 | imprensa |
| GIR, AI washing enforcement | https://globalinvestigationsreview.com/review/the-investigations-review-of-the-americas/2026/article/us-enforcement-agencies-intensify-scrutiny-of-ai-washing | 2026 | imprensa (só snippet) |
| Astella, podcast Lastro | https://www.astella.com.br/matrix/allan-paladino-podcast-astella-playbook | 11/ago/26 | opinião |
| startups.com.br, Rigonatti | https://startups.com.br/negocios/5-minutos-com-edson-rigonatti-socio-e-cofundador-da-astella/ | 06/mar/26 | imprensa |
