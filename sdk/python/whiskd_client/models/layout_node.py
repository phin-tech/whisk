from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="LayoutNode")


@_attrs_define
class LayoutNode:
    """
    Attributes:
        kind (str):
        children (list[LayoutNode] | Unset):
        direction (str | Unset):
        pane_id (str | Unset):
        sizes (list[float] | Unset):
    """

    kind: str
    children: list[LayoutNode] | Unset = UNSET
    direction: str | Unset = UNSET
    pane_id: str | Unset = UNSET
    sizes: list[float] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        kind = self.kind

        children: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.children, Unset):
            children = []
            for children_item_data in self.children:
                children_item = children_item_data.to_dict()
                children.append(children_item)

        direction = self.direction

        pane_id = self.pane_id

        sizes: list[float] | Unset = UNSET
        if not isinstance(self.sizes, Unset):
            sizes = self.sizes

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "kind": kind,
            }
        )
        if children is not UNSET:
            field_dict["children"] = children
        if direction is not UNSET:
            field_dict["direction"] = direction
        if pane_id is not UNSET:
            field_dict["paneId"] = pane_id
        if sizes is not UNSET:
            field_dict["sizes"] = sizes

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = d.pop("kind")

        _children = d.pop("children", UNSET)
        children: list[LayoutNode] | Unset = UNSET
        if _children is not UNSET:
            children = []
            for children_item_data in _children:
                children_item = LayoutNode.from_dict(children_item_data)

                children.append(children_item)

        direction = d.pop("direction", UNSET)

        pane_id = d.pop("paneId", UNSET)

        sizes = cast(list[float], d.pop("sizes", UNSET))

        layout_node = cls(
            kind=kind,
            children=children,
            direction=direction,
            pane_id=pane_id,
            sizes=sizes,
        )

        layout_node.additional_properties = d
        return layout_node

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
