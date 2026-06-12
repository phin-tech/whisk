from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="RunEvent")


@_attrs_define
class RunEvent:
    """
    Attributes:
        at (datetime.datetime):
        id (str):
        type_ (str):
        actor (str | Unset):
        message (str | Unset):
    """

    at: datetime.datetime
    id: str
    type_: str
    actor: str | Unset = UNSET
    message: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        id = self.id

        type_ = self.type_

        actor = self.actor

        message = self.message

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
        if message is not UNSET:
            field_dict["message"] = message

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))

        id = d.pop("id")

        type_ = d.pop("type")

        actor = d.pop("actor", UNSET)

        message = d.pop("message", UNSET)

        run_event = cls(
            at=at,
            id=id,
            type_=type_,
            actor=actor,
            message=message,
        )

        run_event.additional_properties = d
        return run_event

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
