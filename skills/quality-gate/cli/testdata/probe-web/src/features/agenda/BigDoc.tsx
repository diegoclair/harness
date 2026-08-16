/**
 * Renders the day board.
 *
 * The reasoning is spelled out here at length so the block runs past the budget
 * a function doc is allowed to spend, which is the point of the fixture: it has
 * to be long enough to trip CMT-01 and no rule other than CMT-01, so the
 * assertion stays about the budget and nothing else creeps in alongside it,
 * which needs one line more than the budget a function doc is given.
 */
export function BigDoc() {
  return <article />;
}

/**
 * Module-scope binding with a doc four lines long, which is inside the orphan
 * budget: a constant is neither a member nor a contract, and pinning it to the
 * two-line member budget reported every documented constant in the repo.
 */
export const BOARD_COLUMNS = 7;
