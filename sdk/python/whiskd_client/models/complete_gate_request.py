from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="CompleteGateRequest")


@_attrs_define
class CompleteGateRequest:
    """
    Attributes:
        id (str):
        status (str):
        actor (str | Unset):
        override_reason (str | Unset):
    """

    id: str
    status: str
    actor: str | Unset = UNSET
    override_reason: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id = self.id

        status = self.status

        actor = self.actor

        override_reason = self.override_reason

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "id": id,
                "status": status,
            }
        )
        if actor is not UNSET:
            field_dict["actor"] = actor
        if override_reason is not UNSET:
            field_dict["overrideReason"] = override_reason

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        status = d.pop("status")

        actor = d.pop("actor", UNSET)

        override_reason = d.pop("overrideReason", UNSET)

        complete_gate_request = cls(
            id=id,
            status=status,
            actor=actor,
            override_reason=override_reason,
        )

        complete_gate_request.additional_properties = d
        return complete_gate_request

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
