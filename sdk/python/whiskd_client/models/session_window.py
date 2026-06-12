from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.layout_node import LayoutNode


T = TypeVar("T", bound="SessionWindow")


@_attrs_define
class SessionWindow:
    """
    Attributes:
        id (str):
        layout (LayoutNode):
        name (str):
        session_id (str):
    """

    id: str
    layout: LayoutNode
    name: str
    session_id: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id = self.id

        layout = self.layout.to_dict()

        name = self.name

        session_id = self.session_id

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "id": id,
                "layout": layout,
                "name": name,
                "sessionId": session_id,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.layout_node import LayoutNode

        d = dict(src_dict)
        id = d.pop("id")

        layout = LayoutNode.from_dict(d.pop("layout"))

        name = d.pop("name")

        session_id = d.pop("sessionId")

        session_window = cls(
            id=id,
            layout=layout,
            name=name,
            session_id=session_id,
        )

        session_window.additional_properties = d
        return session_window

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
