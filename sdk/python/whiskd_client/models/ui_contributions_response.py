from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.ui_contribution_plugin import UIContributionPlugin
    from ..models.ui_contribution_scope import UIContributionScope


T = TypeVar("T", bound="UIContributionsResponse")


@_attrs_define
class UIContributionsResponse:
    """
    Attributes:
        scope (UIContributionScope):
        plugins (list[UIContributionPlugin] | Unset):
    """

    scope: UIContributionScope
    plugins: list[UIContributionPlugin] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        scope = self.scope.to_dict()

        plugins: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.plugins, Unset):
            plugins = []
            for plugins_item_data in self.plugins:
                plugins_item = plugins_item_data.to_dict()
                plugins.append(plugins_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "scope": scope,
            }
        )
        if plugins is not UNSET:
            field_dict["plugins"] = plugins

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ui_contribution_plugin import UIContributionPlugin
        from ..models.ui_contribution_scope import UIContributionScope

        d = dict(src_dict)
        scope = UIContributionScope.from_dict(d.pop("scope"))

        _plugins = d.pop("plugins", UNSET)
        plugins: list[UIContributionPlugin] | Unset = UNSET
        if _plugins is not UNSET:
            plugins = []
            for plugins_item_data in _plugins:
                plugins_item = UIContributionPlugin.from_dict(plugins_item_data)

                plugins.append(plugins_item)

        ui_contributions_response = cls(
            scope=scope,
            plugins=plugins,
        )

        ui_contributions_response.additional_properties = d
        return ui_contributions_response

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
