from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.item import Item


T = TypeVar("T", bound="OnboardingStatus")


@_attrs_define
class OnboardingStatus:
    """
    Attributes:
        items (list[Item] | None):
        local_daemon (bool):
        should_show (bool):
        state_path (str):
    """

    items: list[Item] | None
    local_daemon: bool
    should_show: bool
    state_path: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        items: list[dict[str, Any]] | None
        if isinstance(self.items, list):
            items = []
            for items_type_0_item_data in self.items:
                items_type_0_item = items_type_0_item_data.to_dict()
                items.append(items_type_0_item)

        else:
            items = self.items

        local_daemon = self.local_daemon

        should_show = self.should_show

        state_path = self.state_path

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "items": items,
                "localDaemon": local_daemon,
                "shouldShow": should_show,
                "statePath": state_path,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.item import Item

        d = dict(src_dict)

        def _parse_items(data: object) -> list[Item] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                items_type_0 = []
                _items_type_0 = data
                for items_type_0_item_data in _items_type_0:
                    items_type_0_item = Item.from_dict(items_type_0_item_data)

                    items_type_0.append(items_type_0_item)

                return items_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[Item] | None, data)

        items = _parse_items(d.pop("items"))

        local_daemon = d.pop("localDaemon")

        should_show = d.pop("shouldShow")

        state_path = d.pop("statePath")

        onboarding_status = cls(
            items=items,
            local_daemon=local_daemon,
            should_show=should_show,
            state_path=state_path,
        )

        onboarding_status.additional_properties = d
        return onboarding_status

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
