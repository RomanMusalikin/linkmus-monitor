import { createContext, useContext } from 'react';

export const NodesContext = createContext(null);

export function useNodesContext() {
  return useContext(NodesContext);
}
