from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="StartExecutionRequest")


@_attrs_define
class StartExecutionRequest:
    """
    Attributes:
        work_item_id (str):
        actor (str | Unset):
        agent_profile_id (str | Unset):
        launch (bool | Unset):
        pty_id (str | Unset):
        session_id (str | Unset):
        system_prompt (str | Unset):
        worktree_override_path (str | Unset):
    """

    work_item_id: str
    actor: str | Unset = UNSET
    agent_profile_id: str | Unset = UNSET
    launch: bool | Unset = UNSET
    pty_id: str | Unset = UNSET
    session_id: str | Unset = UNSET
    system_prompt: str | Unset = UNSET
    worktree_override_path: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        work_item_id = self.work_item_id

        actor = self.actor

        agent_profile_id = self.agent_profile_id

        launch = self.launch

        pty_id = self.pty_id

        session_id = self.session_id

        system_prompt = self.system_prompt

        worktree_override_path = self.worktree_override_path

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "workItemId": work_item_id,
            }
        )
        if actor is not UNSET:
            field_dict["actor"] = actor
        if agent_profile_id is not UNSET:
            field_dict["agentProfileId"] = agent_profile_id
        if launch is not UNSET:
            field_dict["launch"] = launch
        if pty_id is not UNSET:
            field_dict["ptyId"] = pty_id
        if session_id is not UNSET:
            field_dict["sessionId"] = session_id
        if system_prompt is not UNSET:
            field_dict["systemPrompt"] = system_prompt
        if worktree_override_path is not UNSET:
            field_dict["worktreeOverridePath"] = worktree_override_path

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        work_item_id = d.pop("workItemId")

        actor = d.pop("actor", UNSET)

        agent_profile_id = d.pop("agentProfileId", UNSET)

        launch = d.pop("launch", UNSET)

        pty_id = d.pop("ptyId", UNSET)

        session_id = d.pop("sessionId", UNSET)

        system_prompt = d.pop("systemPrompt", UNSET)

        worktree_override_path = d.pop("worktreeOverridePath", UNSET)

        start_execution_request = cls(
            work_item_id=work_item_id,
            actor=actor,
            agent_profile_id=agent_profile_id,
            launch=launch,
            pty_id=pty_id,
            session_id=session_id,
            system_prompt=system_prompt,
            worktree_override_path=worktree_override_path,
        )

        start_execution_request.additional_properties = d
        return start_execution_request

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
