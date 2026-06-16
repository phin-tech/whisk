from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="LaunchWorkItemRunRequest")


@_attrs_define
class LaunchWorkItemRunRequest:
    """
    Attributes:
        id (str):
        actor (str | Unset):
        agent_profile_id (str | Unset):
        system_prompt (str | Unset):
    """

    id: str
    actor: str | Unset = UNSET
    agent_profile_id: str | Unset = UNSET
    system_prompt: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id = self.id

        actor = self.actor

        agent_profile_id = self.agent_profile_id

        system_prompt = self.system_prompt

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "id": id,
            }
        )
        if actor is not UNSET:
            field_dict["actor"] = actor
        if agent_profile_id is not UNSET:
            field_dict["agentProfileId"] = agent_profile_id
        if system_prompt is not UNSET:
            field_dict["systemPrompt"] = system_prompt

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        actor = d.pop("actor", UNSET)

        agent_profile_id = d.pop("agentProfileId", UNSET)

        system_prompt = d.pop("systemPrompt", UNSET)

        launch_work_item_run_request = cls(
            id=id,
            actor=actor,
            agent_profile_id=agent_profile_id,
            system_prompt=system_prompt,
        )

        launch_work_item_run_request.additional_properties = d
        return launch_work_item_run_request

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
