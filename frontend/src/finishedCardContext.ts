import { createContext, useContext } from "react";

/** Opens the finished-card dialog for a game; justFinished titles it "Finished!". */
export type ShowFinishedCard = (appid: number, justFinished?: boolean) => void;

export const FinishedCardCtx = createContext<ShowFinishedCard>(() => {});

export const useShowFinishedCard = () => useContext(FinishedCardCtx);
