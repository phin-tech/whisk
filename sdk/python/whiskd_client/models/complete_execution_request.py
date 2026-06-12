from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="CompleteExecutionRequest")


@_attrs_define
class CompleteExecutionRequest:
    """
    Attributes:
        run_id (str):
        actor (str | Unset):
        message (str | Unset):
        work_item_id (str | Unset):
    """

    run_id: str
    actor: str | Unset = UNSET
    message: str | Unset = UNSET
    work_item_id: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        run_id = self.run_id

        actor = self.actor

        message = self.message

        work_item_id = self.work_item_id

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "runId": run_id,
            }
        )
        if actor is not UNSET:
            field_dict["actor"] = actor
        if message is not UNSET:
            field_dict["message"] = message
        if work_item_id is not UNSET:
            field_dict["workItemId"] = work_item_id

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        run_id = d.pop("runId")

        actor = d.pop("actor", UNSET)

        message = d.pop("message", UNSET)

        work_item_id = d.pop("workItemId", UNSET)

        complete_execution_request = cls(
            run_id=run_id,
            actor=actor,
            message=message,
            work_item_id=work_item_id,
        )

        complete_execution_request.additional_properties = d
        return complete_execution_request

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
