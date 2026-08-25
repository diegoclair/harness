# AI-first — YC, casos com número, FDE, lean AI (ago/2026)

Filtro: só o que contradiz ou acrescenta à skill `business-ai-first` (SKILL.md + §7 de ai-first-2026.md). Confirmação não entra.

**Lido (página inteira):** ycombinator.com/rfs (só edição Fall 2026 no ar) · Extruct W26 · thevccorner (RFS Summer 26, citações) · foundevo (RFS agências) · ai-native-agency.com · Emergence AI-Native Services Playbook · a16z Palantirization · Pragmatic Engineer FDE · Sierra outcome-based (dez/24) e Outcomemaxxing (jun/26) · Contrary/Sierra · Sacra Decagon, Cursor, ElevenLabs, Glean · ZenML Decagon FDE · Quiq Decagon pricing · Metronome Harvey · TechCrunch Lovable (mar/26), Sierra (abr/26), June (ago/26) · businessmodelanalyst Lovable · Vanta/Garry Tan (mai/25) · startuphub Garry Tan (jul/26) · ycrootaccess Garry Tan (ago/26) · techtoken W26 · Henry Shi substack (2 posts) · Every/Armstrong · InfoWorld Crescendo · digitalapplied roll-ups · paulgraham.com/foundermode · Forbes pivôs (jul/26).
**Não lido:** CNBC YC (403) · Inc RFS (403) · Dealroom Cursor (403) · leanaileaderboard.com e GitHub (dados só via JS) · leanaireport.com (TLS) · YC RFS páginas individuais (404) · YC Library "Vertical AI agents" e "Playbook AI native company" (só título) · CNN founder mode (451) · Medium Antin (404) · HN thread agências (sem comentários no HTML) · The Information "Nuances of Cursor's gross margin" (paywall, só snippet) · Henry Shi X posts (snippet).

## §1 Contradições à skill

**1. "Founder demitido continuamente / herói → arquiteto" vs. Founder Mode (PG, set/24, opinião).**
Skill: o founder sai de líder de output pra líder de sistema; cada entrega A→B que precisa dele é herói não sistematizado.
Fonte: "hire good people and give them room to do their jobs" é o conselho que "damaged their companies"; skip-level vira norma; delegar por camadas produz "professional fakers".
Garry Tan (ago/26, opinião) fecha a ponte sem dizer: "never do one-off work" — o founder continua com a mão na massa, mas cada tarefa vira skill file (220k páginas Markdown no GBrain dele).
Recomendo **reescrever**: o founder é demitido do *output repetido*, nunca do *contato direto com o detalhe*. Skip-level em AI-first = ler o log do agente, não o relatório do gerente. Gatilho: se o founder só vê dashboards, o sistema está aprendendo sem ele.

**2. "Mensalidade = cota de trabalho; se falhou não conta" vs. o que o mercado de outcome pricing efetivamente vende.**
Decagon (imprensa, Quiq mar/26 + Sacra): oferece por resolução, mas "vast majority" dos clientes escolhe **por conversa** (~US$ 0,99) + plataforma US$ 50k/ano; Sierra (oficial, jun/26): usa consumo "where it fits the shape of the problem"; Cursor (Sacra, jul/26): passou a token pass-through + ~20%; Lovable (imprensa, ago/26): migrou pra créditos porque "losing money on its heaviest users".
Harvey (Metronome, jan/26, imprensa): **assento** US$ 12–16,8k/ano, uso ilimitado, US$ 190M ARR — o maior vertical AI cobra exatamente o que a skill proíbe.
Recomendo **adicionar gatilho**: cota por resultado é a oferta-âncora, mas a unidade de uso (tarefa tentada) tem que existir como plano paralelo; se >70% dos clientes ficarem no uso, a definição de "resultado" está cara demais de auditar (Quiq: "only works if 'resolution' is consistently defined and measurable"). Assento é aceitável quando o comprador é o operador profissional com uso ilimitado — não é o caso do seller.

**3. "Margem é decisão de produto" vs. margem decidida pelo modelo.**
Cursor: margem bruta **−23%** no trimestre até jan/26 → "slight gross-margin profitability" em abr/26 (Sacra/The Information, imprensa), só porque o Composer próprio absorveu ~metade do autocomplete; contas individuais seguem **loss-making**, enterprise positiva. Lovable, Bolt, Replit, ElevenLabs, Harvey, Glean, Decagon, Sierra: **nenhum** publica margem bruta (não apurado em 8 casos).
Recomendo **adicionar regra**: roteamento de modelo por tarefa (barato → caro só quando aceite cai) é a primeira alavanca de margem, antes de preço; e "margem por segmento": self-serve pequeno é onde a margem fica negativa — medir SMB separado.

**4. "Serviço encolhe com a curva N < N−1" vs. demanda de serviço crescendo.**
TechCrunch (ago/26, imprensa): "AI, paradoxically, increases the demand for professional services" (Rapoport/June); Sierra e Decagon empregam FDEs que "must constantly update and fine-tune customer agents"; onboarding Sierra 4–10 semanas; Decagon 50→500 funcionários em 1 ano.
Emergence (mar/26, opinião): "staff pilots with a dedicated team", "over-invest in migration" — humano pesado no início, de propósito.
Recomendo **manter** a curva, mas **adicionar gatilho numérico**: Decagon produtizou integração "after approximately the 25th custom integration"; Crosby Legal passou a dizer não a custom "at approximately 80 clients". Regra: até ~25 entregas o serviço é pesquisa; se na 25ª ainda não houve template, é software house.

**5. "One-person unicorn" (Altman, fev/24, opinião) vs. Every/Armstrong (fev/24, opinião): "not possible" a US$ 1B; janela real é US$ 1–10M ARR solo** — distribuição e hand-holding enterprise não se fazem sozinho. Skill não promete unicórnio, mas cita "OS one-person"; **manter**, carimbando o alvo solo como US$ 1–10M.

## §2 Acréscimos

- **Mirage PMF** (Emergence, mar/26, opinião): receita e retenção não provam PMF em AI-native services; "you only truly have it when AI is doing a material share". Sinais de miragem: margem bruta flat/caindo, receita/funcionário estagnada, headcount linear. → entra no passo 1 do ritual (estágio) como teste de "PMF de verdade".
- **HURT — Human Review Time em minutos** (Crosby via Emergence) como north star de produto; complementa "% humano" da skill com **tempo**, não só contagem.
- **Inferência dentro do COGS "honestly"** (Emergence) e **margem por cliente** — já na skill; acrescenta "revenue per employee" como lagging.
- **"Automate tasks, not people"** e "hire product leader earlier than intuition suggests" (Emergence).
- **"It's the demo, stupid"** — demo da IA fazendo o trabalho corta o ciclo de venda pela metade (Emergence, sem número de base).
- **YC W26** (Extruct, mar/26, estudo): 199 empresas, **28% AI-native service**, 22% AI-enhanced software, 11% solo founder; 14 com US$ 1M ARR (3x W25); **14% WoW** médio (techtoken, imprensa). YC já classifica "serviço AI-native" como categoria própria e majoritária.
- **RFS Summer 26** (thevccorner, imprensa): "The next era is companies that skip the human entirely and just do the work" (AI-Native Service Companies, Alströmer); "Company Brain" (Blomfield) = "a living map of how a company actually works" com skills executáveis; SaaS Challengers: "collapsed the cost of producing software by 10 to 100x". RFS Spring 26: AI-Native Agencies (Migicovsky/Epstein): "sells finished deliverables at premium prices", "margins approach software levels while retaining service business pricing power"; humano sênior fica com "strategy, exceptions, and accountability". Margem 65–80% citada só em site terceiro (ai-native-agency.com, opinião), não no texto da YC.
- **FDE de origem** (Pragmatic Engineer, ago/25, imprensa): "one customer, many capabilities" vs. dev tradicional "one capability, many customers"; até ~2016 Palantir tinha mais FDE que engenheiro de produto; FDE ~25% do tempo on-site. a16z (jan/26): falha em "10–20% efficiency gains", "thousands of small accounts", "snowflake customers"; "seven-figure deals easier than five-figure ones".
- **Roll-up de agências** (digitalapplied, mai/26, opinião, cálculo próprio): margem bruta agência ~30% → ~50% pós-agentic; múltiplo 0,7–1,1x receita; 21 deals H1/26 (+162%). Crescendo: margem 4x call center (10–15%), >90% resolvido por IA após 1º mês, US$ 2,99/resolução (InfoWorld set/24 + G2, imprensa).
- **Lean AI** (Henry Shi, ago/25, opinião): receita/funcionário US$ 1–5M; tempo até US$ 5M ARR 9 meses vs 24 tradicional; "US$ 1M de receita com US$ 200k de gasto". Lovable US$ 2,77M ARR/funcionário, 146 pessoas, US$ 400M ARR (TechCrunch mar/26, imprensa). Retell US$ 60M/~40 pessoas (Garry Tan, ago/26, opinião). Top-10 média US$ 7,3M/funcionário (X, snippet — não lido).
- **Garry Tan**: 25% do batch W25 com 95% do código por IA (mai/25); "the leverage is not in the weights, it's in how you wire the work" (jul/26); "skillify" tudo = versão YC do company brain da skill.

## §3 Casos com número

| Empresa | Cobrança | Margem bruta | % humano | Fonte | Data | Carimbo |
|---|---|---|---|---|---|---|
| Sierra | por resolução (~US$ 1,50 snippet); consumo onde cabe; FDE + onboarding 4–10 sem | não apurado (~70% em §7 da skill) | Ramp 90% autônomo; outros 70% | Contrary; sierra.ai; TechCrunch | abr–jun/26 | imprensa/oficial |
| Decagon | US$ 0,99/conversa **ou** por resolução (maioria: conversa) + US$ 50k plataforma; FDE + APM | não apurado | cliente: 3,4% "ask for human"; outro 90% resolvido | Sacra; Quiq; ZenML | mar–jul/26 | imprensa |
| Harvey | assento US$ 12–16,8k/ano, ilimitado; CS "forward-deployed" ex-advogados | não apurado | não apurado | Metronome | jan/26 | imprensa |
| Cursor | assento US$ 20–200 + tokens (API +~20%) | −23% (jan/26) → levemente positiva (abr/26); indivíduo negativo | n/a | Sacra; The Information (snippet) | abr–jul/26 | imprensa |
| Lovable | créditos por complexidade; US$ 25–50/mês | não publicada; créditos pra conter perda em heavy users | n/a | TechCrunch; businessmodelanalyst | mar–ago/26 | imprensa |
| ElevenLabs | uso (caracteres/minutos); ~50/50 self-serve/enterprise; US$ 600M ARR jun/26 | não apurado | n/a | Sacra | jun/26 | imprensa |
| Glean | assento US$ 45–50 + add-on US$ 15; US$ 300M ARR mai/26 | não apurado | n/a | Sacra | mai/26 | imprensa |
| Crescendo | US$ 2,99/resolução | 4x call center (base 10–15%) | ~10% após 1º mês | InfoWorld; G2 | set/24 | imprensa |
| Harper (AINS) | serviço; 5.000 negócios em 13 meses | alvo Emergence ≥50% | não apurado | Emergence | mar/26 | opinião |

Startups AI → serviço/software house: **não apurado com nome** — Forbes (jul/26) lista pivôs de produto (Tome→Lightfield, Wispr, Patronus), nenhum pra serviço; a única evidência é estrutural (FDE em Sierra/Decagon/Harvey; June nasce pra vender implantação).

## §4 Fontes

| Fonte | URL | Data | Carimbo |
|---|---|---|---|
| YC RFS (Fall 26 no ar) | https://www.ycombinator.com/rfs | ago/26 | oficial |
| thevccorner, RFS Summer 26 | https://www.thevccorner.com/p/yc-summer-2026-requests-for-startups-ideas | 2026 | imprensa |
| foundevo, RFS agências | https://www.foundevo.com/ycombinator-requests-for-startups/ | 2026 | imprensa |
| ai-native-agency.com | https://ai-native-agency.com/blog/yc-ai-native-agency | 10/jun/26 | opinião |
| Extruct, YC W26 | https://www.extruct.ai/research/ycw26/ | 25/mar/26 | estudo |
| techtoken, W26 Demo Day | https://techtoken.in/y-combinators-w26-demo-day/ | mar/26 | imprensa |
| Vanta, Garry Tan | https://www.vanta.com/resources/why-the-next-unicorns-are-built-by-ai | 21/mai/25 | opinião |
| startuphub, Garry Tan keynote | https://www.startuphub.ai/ai-news/artificial-intelligence/2026/garry-tan-build-ai-native-companies-not-just-ai-users | 17/jul/26 | imprensa |
| Root Access, Garry Tan | https://www.ycrootaccess.com/p/garry-tan-own-your-intelligence | 06/ago/26 | opinião |
| Paul Graham, Founder Mode | https://paulgraham.com/foundermode.html | set/24 | opinião |
| Emergence, AINS Playbook | https://www.emcap.com/thoughts/the-ai-native-services-playbook | 30/mar/26 | opinião |
| a16z, Palantirization | https://a16z.com/the-palantirization-of-everything/ | 16/jan/26 | opinião |
| Pragmatic Engineer, FDE | https://newsletter.pragmaticengineer.com/p/forward-deployed-engineers | 12/ago/25 | imprensa |
| Sierra, outcome pricing | https://sierra.ai/blog/outcome-based-pricing-for-ai-agents | 10/dez/24 | oficial |
| Sierra, Outcomemaxxing | https://sierra.ai/blog/outcomemaxxing | 03/jun/26 | oficial |
| Contrary, Sierra | https://research.contrary.com/company/sierra | 2026 | imprensa |
| TechCrunch, Sierra/Taylor | https://techcrunch.com/2026/04/09/sierras-bret-taylor-says-the-era-of-clicking-buttons-is-over/ | 09/abr/26 | imprensa |
| Sacra, Decagon | https://sacra.com/c/decagon/ | jul/26 | imprensa |
| Quiq, Decagon pricing | https://quiq.com/blog/decagon-pricing/ | 11/mar/26 | imprensa (concorrente) |
| ZenML, Decagon FDE | https://www.zenml.io/llmops-database/scaling-forward-deployment-engineering-for-ai-customer-service-agents | 2026 | imprensa |
| Metronome, Harvey | https://metronome.com/pricing-index/harvey | 23/jan/26 | imprensa |
| Sacra, Cursor | https://sacra.com/c/cursor/ | jul/26 | imprensa |
| Value Add VC, Cursor | https://valueaddvc.com/blog/how-does-cursor-make-money-subscriptions-token-pricing-and-the-business-model-breakdown | 22/jul/26 | opinião |
| TechCrunch, Lovable | https://techcrunch.com/2026/03/11/lovable-says-it-added-100m-in-revenue-last-month-alone-with-just-146-employees/ | 11/mar/26 | imprensa |
| businessmodelanalyst, Lovable | https://businessmodelanalyst.com/lovable-series-c-cost-of-goods-capital/ | 13/ago/26 | opinião |
| Sacra, ElevenLabs | https://sacra.com/c/elevenlabs/ | jun/26 | imprensa |
| Sacra, Glean | https://sacra.com/research/glean-at-200m-arr/ | nov/25 | imprensa |
| TechCrunch, June/Rapoport | https://techcrunch.com/2026/08/03/a-marc-benioff-backed-startup-thinks-ai-can-solve-the-ai-deployment-problem/ | 03/ago/26 | imprensa |
| InfoWorld, Crescendo | https://www.infoworld.com/article/3542335/crescendo-makes-ai-boring-and-profitable.html | 30/set/24 | imprensa |
| digitalapplied, roll-ups | https://www.digitalapplied.com/blog/ai-agency-rollup-wave-m-and-a-predictions-2026 | 01/mai/26 | opinião |
| Henry Shi, apresentação | https://henrythe9th.substack.com/p/my-presentation-on-lean-ai-native | 28/ago/25 | opinião |
| Henry Shi, leaderboard launch | https://henrythe9th.substack.com/p/launching-the-official-lean-ai-leaderboard | 25/mar/25 | opinião (parcial, paywall) |
| Every, Armstrong | https://every.to/napkin-math/the-one-person-billion-dollar-company | 07/fev/24 | opinião |
| Forbes, pivôs | https://www.forbes.com/sites/rashishrivastava/2026/07/20/ai-startups-are-pivoting-from-flashy-demos-to-tech-that-pays-the-bills/ | 20/jul/26 | imprensa |
| Não lidos | CNBC 15/mar/25 (403) · Inc RFS (403) · Dealroom Cursor (403) · The Information Cursor abr/26 (paywall) · leanaileaderboard.com (JS) · leanaireport.com (TLS) · CNN founder mode (451) · YC Library ×2 (só título) | — | — |
