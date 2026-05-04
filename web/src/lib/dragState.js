// Synchronous module-level flag — set before React state changes
// so onClick handlers can block navigation immediately after drag
export const dragState = { happened: false };
