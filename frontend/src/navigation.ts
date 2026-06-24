export type MainView = "session" | "work" | "projects";

export type NavigationState = {
  activeMain: MainView;
  navigationStack: MainView[];
  workBoardOpenItemId: string;
};

export function navigateTo(
  state: NavigationState,
  target: MainView,
  opts?: { openItemId?: string },
): NavigationState {
  const navigationStack =
    state.activeMain === target ? state.navigationStack : [...state.navigationStack, state.activeMain];
  return {
    activeMain: target,
    navigationStack,
    workBoardOpenItemId:
      opts?.openItemId === undefined ? state.workBoardOpenItemId : opts.openItemId,
  };
}

export function navigateBack(state: NavigationState): NavigationState {
  const prev = state.navigationStack.at(-1);
  return {
    activeMain: prev ?? state.activeMain,
    navigationStack: state.navigationStack.slice(0, -1),
    workBoardOpenItemId: "",
  };
}

export function clearNavigationStack(state: NavigationState): NavigationState {
  return {
    ...state,
    navigationStack: [],
    workBoardOpenItemId: "",
  };
}
