from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="MoveWorkItemRequest")


@_attrs_define
class MoveWorkItemRequest:
    """
    Attributes:
        id (str):
        stage_id (str):
        actor (str | Unset):
    """

    id: str
    stage_id: str
    actor: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id = self.id

        stage_id = self.stage_id

        actor = self.actor

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "id": id,
                "stageId": stage_id,
            }
        )
        if actor is not UNSET:
            field_dict["actor"] = actor

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        stage_id = d.pop("stageId")

        actor = d.pop("actor", UNSET)

        move_work_item_request = cls(
            id=id,
            stage_id=stage_id,
            actor=actor,
        )

        move_work_item_request.additional_properties = d
        return move_work_item_request

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
