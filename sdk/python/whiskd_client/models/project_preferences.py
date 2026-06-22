from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.gate_config import GateConfig
    from ..models.project_preferences_default_phase_agents import (
        ProjectPreferencesDefaultPhaseAgents,
    )


T = TypeVar("T", bound="ProjectPreferences")


@_attrs_define
class ProjectPreferences:
    """
    Attributes:
        auto_run (str):
        auto_worktree (bool):
        default_phase_agents (ProjectPreferencesDefaultPhaseAgents | Unset):
        gates (list[GateConfig] | Unset):
        use_interactive_agent_shell (bool | Unset):
    """

    auto_run: str
    auto_worktree: bool
    default_phase_agents: ProjectPreferencesDefaultPhaseAgents | Unset = UNSET
    gates: list[GateConfig] | Unset = UNSET
    use_interactive_agent_shell: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        auto_run = self.auto_run

        auto_worktree = self.auto_worktree

        default_phase_agents: dict[str, Any] | Unset = UNSET
        if not isinstance(self.default_phase_agents, Unset):
            default_phase_agents = self.default_phase_agents.to_dict()

        gates: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.gates, Unset):
            gates = []
            for gates_item_data in self.gates:
                gates_item = gates_item_data.to_dict()
                gates.append(gates_item)

        use_interactive_agent_shell = self.use_interactive_agent_shell

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "autoRun": auto_run,
                "autoWorktree": auto_worktree,
            }
        )
        if default_phase_agents is not UNSET:
            field_dict["defaultPhaseAgents"] = default_phase_agents
        if gates is not UNSET:
            field_dict["gates"] = gates
        if use_interactive_agent_shell is not UNSET:
            field_dict["useInteractiveAgentShell"] = use_interactive_agent_shell

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.gate_config import GateConfig
        from ..models.project_preferences_default_phase_agents import (
            ProjectPreferencesDefaultPhaseAgents,
        )

        d = dict(src_dict)
        auto_run = d.pop("autoRun")

        auto_worktree = d.pop("autoWorktree")

        _default_phase_agents = d.pop("defaultPhaseAgents", UNSET)
        default_phase_agents: ProjectPreferencesDefaultPhaseAgents | Unset
        if isinstance(_default_phase_agents, Unset):
            default_phase_agents = UNSET
        else:
            default_phase_agents = ProjectPreferencesDefaultPhaseAgents.from_dict(
                _default_phase_agents
            )

        _gates = d.pop("gates", UNSET)
        gates: list[GateConfig] | Unset = UNSET
        if _gates is not UNSET:
            gates = []
            for gates_item_data in _gates:
                gates_item = GateConfig.from_dict(gates_item_data)

                gates.append(gates_item)

        use_interactive_agent_shell = d.pop("useInteractiveAgentShell", UNSET)

        project_preferences = cls(
            auto_run=auto_run,
            auto_worktree=auto_worktree,
            default_phase_agents=default_phase_agents,
            gates=gates,
            use_interactive_agent_shell=use_interactive_agent_shell,
        )

        project_preferences.additional_properties = d
        return project_preferences

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
