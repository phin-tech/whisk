from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.worktrunk_binary import WorktrunkBinary


T = TypeVar("T", bound="WorktrunkStatus")


@_attrs_define
class WorktrunkStatus:
    """
    Attributes:
        available (bool):
        binary (WorktrunkBinary):
        config_found (bool):
    """

    available: bool
    binary: WorktrunkBinary
    config_found: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        available = self.available

        binary = self.binary.to_dict()

        config_found = self.config_found

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "available": available,
                "binary": binary,
                "configFound": config_found,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.worktrunk_binary import WorktrunkBinary

        d = dict(src_dict)
        available = d.pop("available")

        binary = WorktrunkBinary.from_dict(d.pop("binary"))

        config_found = d.pop("configFound")

        worktrunk_status = cls(
            available=available,
            binary=binary,
            config_found=config_found,
        )

        worktrunk_status.additional_properties = d
        return worktrunk_status

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
