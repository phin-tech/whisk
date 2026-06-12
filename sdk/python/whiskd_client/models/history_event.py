from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="HistoryEvent")


@_attrs_define
class HistoryEvent:
    """
    Attributes:
        at (datetime.datetime):
        id (str):
        type_ (str):
        actor (str | Unset):
        attachment_id (str | Unset):
        branch (str | Unset):
        message (str | Unset):
        stage_id (str | Unset):
        worktree_path (str | Unset):
    """

    at: datetime.datetime
    id: str
    type_: str
    actor: str | Unset = UNSET
    attachment_id: str | Unset = UNSET
    branch: str | Unset = UNSET
    message: str | Unset = UNSET
    stage_id: str | Unset = UNSET
    worktree_path: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        id = self.id

        type_ = self.type_

        actor = self.actor

        attachment_id = self.attachment_id

        branch = self.branch

        message = self.message

        stage_id = self.stage_id

        worktree_path = self.worktree_path

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "at": at,
                "id": id,
                "type": type_,
            }
        )
        if actor is not UNSET:
            field_dict["actor"] = actor
        if attachment_id is not UNSET:
            field_dict["attachmentId"] = attachment_id
        if branch is not UNSET:
            field_dict["branch"] = branch
        if message is not UNSET:
            field_dict["message"] = message
        if stage_id is not UNSET:
            field_dict["stageId"] = stage_id
        if worktree_path is not UNSET:
            field_dict["worktreePath"] = worktree_path

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))

        id = d.pop("id")

        type_ = d.pop("type")

        actor = d.pop("actor", UNSET)

        attachment_id = d.pop("attachmentId", UNSET)

        branch = d.pop("branch", UNSET)

        message = d.pop("message", UNSET)

        stage_id = d.pop("stageId", UNSET)

        worktree_path = d.pop("worktreePath", UNSET)

        history_event = cls(
            at=at,
            id=id,
            type_=type_,
            actor=actor,
            attachment_id=attachment_id,
            branch=branch,
            message=message,
            stage_id=stage_id,
            worktree_path=worktree_path,
        )

        history_event.additional_properties = d
        return history_event

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
