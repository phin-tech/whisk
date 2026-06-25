import { describe, expect, it } from "vitest";
import {
  clearNavigationStack,
  navigateBack,
  navigateTo,
  selectMainView,
  type NavigationState,
} from "./navigation";

describe("navigation state", () => {
  it("opens a work item from projects with projects as the return target", () => {
    const state: NavigationState = {
      activeMain: "projects",
      navigationStack: [],
      workBoardOpenItemId: "",
    };

    expect(navigateTo(state, "work", { openItemId: "item-1" })).toEqual({
      activeMain: "work",
      navigationStack: ["projects"],
      workBoardOpenItemId: "item-1",
    });
  });

  it("returns from a deep-linked work item to projects and clears the open item", () => {
    const state: NavigationState = {
      activeMain: "work",
      navigationStack: ["projects"],
      workBoardOpenItemId: "item-1",
    };

    expect(navigateBack(state)).toEqual({
      activeMain: "projects",
      navigationStack: [],
      workBoardOpenItemId: "",
    });
  });

  it("clears return context without changing the active main view", () => {
    const state: NavigationState = {
      activeMain: "work",
      navigationStack: ["projects"],
      workBoardOpenItemId: "item-1",
    };

    expect(clearNavigationStack(state)).toEqual({
      activeMain: "work",
      navigationStack: [],
      workBoardOpenItemId: "",
    });
  });

  it("replaces same-target work detail without pushing a duplicate return target", () => {
    const state: NavigationState = {
      activeMain: "work",
      navigationStack: ["projects"],
      workBoardOpenItemId: "item-1",
    };

    expect(navigateTo(state, "work", { openItemId: "item-2" })).toEqual({
      activeMain: "work",
      navigationStack: ["projects"],
      workBoardOpenItemId: "item-2",
    });
  });

  it("leaves the active main view unchanged when navigating back with an empty stack", () => {
    const state: NavigationState = {
      activeMain: "work",
      navigationStack: [],
      workBoardOpenItemId: "item-1",
    };

    expect(navigateBack(state)).toEqual({
      activeMain: "work",
      navigationStack: [],
      workBoardOpenItemId: "",
    });
  });

  it("selects a root main view and clears deep-link return context", () => {
    const state: NavigationState = {
      activeMain: "work",
      navigationStack: ["projects"],
      workBoardOpenItemId: "item-1",
    };

    expect(selectMainView(state, "session")).toEqual({
      activeMain: "session",
      navigationStack: [],
      workBoardOpenItemId: "",
    });
  });
});
