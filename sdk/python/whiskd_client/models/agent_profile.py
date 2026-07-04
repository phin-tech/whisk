from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="AgentProfile")


@_attrs_define
class AgentProfile:
    """
    Attributes:
        id (str):
        label (str):
        launchable (bool):
        prompt_injection_mode (str):
        provider (str):
        source (str):
        description (str | Unset):
        detect_aliases (list[str] | Unset):
        detect_cmd (str | Unset):
        draft_prompt_env_var (str | Unset):
        draft_prompt_flag (str | Unset):
        expected_process (str | Unset):
        hook_provider (str | Unset):
        launch_blocked_reason (str | Unset):
        plugin_id (str | Unset):
        preflight_trust (str | Unset):
        ready_signal (str | Unset):
    """

    id: str
    label: str
    launchable: bool
    prompt_injection_mode: str
    provider: str
    source: str
    description: str | Unset = UNSET
    detect_aliases: list[str] | Unset = UNSET
    detect_cmd: str | Unset = UNSET
    draft_prompt_env_var: str | Unset = UNSET
    draft_prompt_flag: str | Unset = UNSET
    expected_process: str | Unset = UNSET
    hook_provider: str | Unset = UNSET
    launch_blocked_reason: str | Unset = UNSET
    plugin_id: str | Unset = UNSET
    preflight_trust: str | Unset = UNSET
    ready_signal: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id = self.id

        label = self.label

        launchable = self.launchable

        prompt_injection_mode = self.prompt_injection_mode

        provider = self.provider

        source = self.source

        description = self.description

        detect_aliases: list[str] | Unset = UNSET
        if not isinstance(self.detect_aliases, Unset):
            detect_aliases = self.detect_aliases

        detect_cmd = self.detect_cmd

        draft_prompt_env_var = self.draft_prompt_env_var

        draft_prompt_flag = self.draft_prompt_flag

        expected_process = self.expected_process

        hook_provider = self.hook_provider

        launch_blocked_reason = self.launch_blocked_reason

        plugin_id = self.plugin_id

        preflight_trust = self.preflight_trust

        ready_signal = self.ready_signal

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "id": id,
                "label": label,
                "launchable": launchable,
                "promptInjectionMode": prompt_injection_mode,
                "provider": provider,
                "source": source,
            }
        )
        if description is not UNSET:
            field_dict["description"] = description
        if detect_aliases is not UNSET:
            field_dict["detectAliases"] = detect_aliases
        if detect_cmd is not UNSET:
            field_dict["detectCmd"] = detect_cmd
        if draft_prompt_env_var is not UNSET:
            field_dict["draftPromptEnvVar"] = draft_prompt_env_var
        if draft_prompt_flag is not UNSET:
            field_dict["draftPromptFlag"] = draft_prompt_flag
        if expected_process is not UNSET:
            field_dict["expectedProcess"] = expected_process
        if hook_provider is not UNSET:
            field_dict["hookProvider"] = hook_provider
        if launch_blocked_reason is not UNSET:
            field_dict["launchBlockedReason"] = launch_blocked_reason
        if plugin_id is not UNSET:
            field_dict["pluginId"] = plugin_id
        if preflight_trust is not UNSET:
            field_dict["preflightTrust"] = preflight_trust
        if ready_signal is not UNSET:
            field_dict["readySignal"] = ready_signal

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        label = d.pop("label")

        launchable = d.pop("launchable")

        prompt_injection_mode = d.pop("promptInjectionMode")

        provider = d.pop("provider")

        source = d.pop("source")

        description = d.pop("description", UNSET)

        detect_aliases = cast(list[str], d.pop("detectAliases", UNSET))

        detect_cmd = d.pop("detectCmd", UNSET)

        draft_prompt_env_var = d.pop("draftPromptEnvVar", UNSET)

        draft_prompt_flag = d.pop("draftPromptFlag", UNSET)

        expected_process = d.pop("expectedProcess", UNSET)

        hook_provider = d.pop("hookProvider", UNSET)

        launch_blocked_reason = d.pop("launchBlockedReason", UNSET)

        plugin_id = d.pop("pluginId", UNSET)

        preflight_trust = d.pop("preflightTrust", UNSET)

        ready_signal = d.pop("readySignal", UNSET)

        agent_profile = cls(
            id=id,
            label=label,
            launchable=launchable,
            prompt_injection_mode=prompt_injection_mode,
            provider=provider,
            source=source,
            description=description,
            detect_aliases=detect_aliases,
            detect_cmd=detect_cmd,
            draft_prompt_env_var=draft_prompt_env_var,
            draft_prompt_flag=draft_prompt_flag,
            expected_process=expected_process,
            hook_provider=hook_provider,
            launch_blocked_reason=launch_blocked_reason,
            plugin_id=plugin_id,
            preflight_trust=preflight_trust,
            ready_signal=ready_signal,
        )

        agent_profile.additional_properties = d
        return agent_profile

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
