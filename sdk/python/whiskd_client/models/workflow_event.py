from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="WorkflowEvent")


@_attrs_define
class WorkflowEvent:
    """
    Attributes:
        at (datetime.datetime):
        id (str):
        project_id (str):
        type_ (str):
        actor (str | Unset):
        message (str | Unset):
        run_id (str | Unset):
        work_item_id (str | Unset):
    """

    at: datetime.datetime
    id: str
    project_id: str
    type_: str
    actor: str | Unset = UNSET
    message: str | Unset = UNSET
    run_id: str | Unset = UNSET
    work_item_id: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        id = self.id

        project_id = self.project_id

        type_ = self.type_

        actor = self.actor

        message = self.message

        run_id = self.run_id

        work_item_id = self.work_item_id

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "at": at,
                "id": id,
                "projectId": project_id,
                "type": type_,
            }
        )
        if actor is not UNSET:
            field_dict["actor"] = actor
        if message is not UNSET:
            field_dict["message"] = message
        if run_id is not UNSET:
            field_dict["runId"] = run_id
        if work_item_id is not UNSET:
            field_dict["workItemId"] = work_item_id

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))

        id = d.pop("id")

        project_id = d.pop("projectId")

        type_ = d.pop("type")

        actor = d.pop("actor", UNSET)

        message = d.pop("message", UNSET)

        run_id = d.pop("runId", UNSET)

        work_item_id = d.pop("workItemId", UNSET)

        workflow_event = cls(
            at=at,
            id=id,
            project_id=project_id,
            type_=type_,
            actor=actor,
            message=message,
            run_id=run_id,
            work_item_id=work_item_id,
        )

        workflow_event.additional_properties = d
        return workflow_event

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
