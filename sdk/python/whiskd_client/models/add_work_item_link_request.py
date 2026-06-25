from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="AddWorkItemLinkRequest")


@_attrs_define
class AddWorkItemLinkRequest:
    """
    Attributes:
        source_work_item_id (str):
        target_work_item_id (str):
        type_ (str):
        actor (str | Unset):
    """

    source_work_item_id: str
    target_work_item_id: str
    type_: str
    actor: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        source_work_item_id = self.source_work_item_id

        target_work_item_id = self.target_work_item_id

        type_ = self.type_

        actor = self.actor

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "sourceWorkItemId": source_work_item_id,
                "targetWorkItemId": target_work_item_id,
                "type": type_,
            }
        )
        if actor is not UNSET:
            field_dict["actor"] = actor

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        source_work_item_id = d.pop("sourceWorkItemId")

        target_work_item_id = d.pop("targetWorkItemId")

        type_ = d.pop("type")

        actor = d.pop("actor", UNSET)

        add_work_item_link_request = cls(
            source_work_item_id=source_work_item_id,
            target_work_item_id=target_work_item_id,
            type_=type_,
            actor=actor,
        )

        add_work_item_link_request.additional_properties = d
        return add_work_item_link_request

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
