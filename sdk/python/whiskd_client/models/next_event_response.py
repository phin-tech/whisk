from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.runtime_event import RuntimeEvent


T = TypeVar("T", bound="NextEventResponse")


@_attrs_define
class NextEventResponse:
    """
    Attributes:
        event (RuntimeEvent):
        missed (bool):
    """

    event: RuntimeEvent
    missed: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        event = self.event.to_dict()

        missed = self.missed

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "event": event,
                "missed": missed,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.runtime_event import RuntimeEvent

        d = dict(src_dict)
        event = RuntimeEvent.from_dict(d.pop("event"))

        missed = d.pop("missed")

        next_event_response = cls(
            event=event,
            missed=missed,
        )

        next_event_response.additional_properties = d
        return next_event_response

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
