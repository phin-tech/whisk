from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.workflow_migration_item import WorkflowMigrationItem


T = TypeVar("T", bound="WorkflowMigrationPlan")


@_attrs_define
class WorkflowMigrationPlan:
    """
    Attributes:
        compatible_items (int):
        current_id (str):
        current_version (int):
        existing_items (int):
        incompatible_items (int):
        items (list[WorkflowMigrationItem] | None):
        items_pinned_to_current_version (int):
        project_id (str):
        target_id (str):
        target_version (int):
    """

    compatible_items: int
    current_id: str
    current_version: int
    existing_items: int
    incompatible_items: int
    items: list[WorkflowMigrationItem] | None
    items_pinned_to_current_version: int
    project_id: str
    target_id: str
    target_version: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        compatible_items = self.compatible_items

        current_id = self.current_id

        current_version = self.current_version

        existing_items = self.existing_items

        incompatible_items = self.incompatible_items

        items: list[dict[str, Any]] | None
        if isinstance(self.items, list):
            items = []
            for items_type_0_item_data in self.items:
                items_type_0_item = items_type_0_item_data.to_dict()
                items.append(items_type_0_item)

        else:
            items = self.items

        items_pinned_to_current_version = self.items_pinned_to_current_version

        project_id = self.project_id

        target_id = self.target_id

        target_version = self.target_version

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "compatibleItems": compatible_items,
                "currentId": current_id,
                "currentVersion": current_version,
                "existingItems": existing_items,
                "incompatibleItems": incompatible_items,
                "items": items,
                "itemsPinnedToCurrentVersion": items_pinned_to_current_version,
                "projectId": project_id,
                "targetId": target_id,
                "targetVersion": target_version,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.workflow_migration_item import WorkflowMigrationItem

        d = dict(src_dict)
        compatible_items = d.pop("compatibleItems")

        current_id = d.pop("currentId")

        current_version = d.pop("currentVersion")

        existing_items = d.pop("existingItems")

        incompatible_items = d.pop("incompatibleItems")

        def _parse_items(data: object) -> list[WorkflowMigrationItem] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                items_type_0 = []
                _items_type_0 = data
                for items_type_0_item_data in _items_type_0:
                    items_type_0_item = WorkflowMigrationItem.from_dict(
                        items_type_0_item_data
                    )

                    items_type_0.append(items_type_0_item)

                return items_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[WorkflowMigrationItem] | None, data)

        items = _parse_items(d.pop("items"))

        items_pinned_to_current_version = d.pop("itemsPinnedToCurrentVersion")

        project_id = d.pop("projectId")

        target_id = d.pop("targetId")

        target_version = d.pop("targetVersion")

        workflow_migration_plan = cls(
            compatible_items=compatible_items,
            current_id=current_id,
            current_version=current_version,
            existing_items=existing_items,
            incompatible_items=incompatible_items,
            items=items,
            items_pinned_to_current_version=items_pinned_to_current_version,
            project_id=project_id,
            target_id=target_id,
            target_version=target_version,
        )

        workflow_migration_plan.additional_properties = d
        return workflow_migration_plan

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
