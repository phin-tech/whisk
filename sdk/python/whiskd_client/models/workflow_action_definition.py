from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.workflow_artifact_effect import WorkflowArtifactEffect
    from ..models.workflow_artifact_requirement import WorkflowArtifactRequirement
    from ..models.workflow_run_effect import WorkflowRunEffect


T = TypeVar("T", bound="WorkflowActionDefinition")


@_attrs_define
class WorkflowActionDefinition:
    """
    Attributes:
        from_ (list[str] | None):
        id (str):
        to (str):
        completes_run (bool | Unset):
        creates_artifact (None | Unset | WorkflowArtifactEffect):
        creates_gates (list[str] | Unset):
        creates_run (None | Unset | WorkflowRunEffect):
        requires (list[WorkflowArtifactRequirement] | Unset):
        requires_human (bool | Unset):
        requires_passing_blocking_gates (bool | Unset):
        resumes_run (str | Unset):
        side_stage (bool | Unset):
        updates_artifact (None | Unset | WorkflowArtifactEffect):
    """

    from_: list[str] | None
    id: str
    to: str
    completes_run: bool | Unset = UNSET
    creates_artifact: None | Unset | WorkflowArtifactEffect = UNSET
    creates_gates: list[str] | Unset = UNSET
    creates_run: None | Unset | WorkflowRunEffect = UNSET
    requires: list[WorkflowArtifactRequirement] | Unset = UNSET
    requires_human: bool | Unset = UNSET
    requires_passing_blocking_gates: bool | Unset = UNSET
    resumes_run: str | Unset = UNSET
    side_stage: bool | Unset = UNSET
    updates_artifact: None | Unset | WorkflowArtifactEffect = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.workflow_artifact_effect import WorkflowArtifactEffect
        from ..models.workflow_run_effect import WorkflowRunEffect

        from_: list[str] | None
        if isinstance(self.from_, list):
            from_ = self.from_

        else:
            from_ = self.from_

        id = self.id

        to = self.to

        completes_run = self.completes_run

        creates_artifact: dict[str, Any] | None | Unset
        if isinstance(self.creates_artifact, Unset):
            creates_artifact = UNSET
        elif isinstance(self.creates_artifact, WorkflowArtifactEffect):
            creates_artifact = self.creates_artifact.to_dict()
        else:
            creates_artifact = self.creates_artifact

        creates_gates: list[str] | Unset = UNSET
        if not isinstance(self.creates_gates, Unset):
            creates_gates = self.creates_gates

        creates_run: dict[str, Any] | None | Unset
        if isinstance(self.creates_run, Unset):
            creates_run = UNSET
        elif isinstance(self.creates_run, WorkflowRunEffect):
            creates_run = self.creates_run.to_dict()
        else:
            creates_run = self.creates_run

        requires: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.requires, Unset):
            requires = []
            for requires_item_data in self.requires:
                requires_item = requires_item_data.to_dict()
                requires.append(requires_item)

        requires_human = self.requires_human

        requires_passing_blocking_gates = self.requires_passing_blocking_gates

        resumes_run = self.resumes_run

        side_stage = self.side_stage

        updates_artifact: dict[str, Any] | None | Unset
        if isinstance(self.updates_artifact, Unset):
            updates_artifact = UNSET
        elif isinstance(self.updates_artifact, WorkflowArtifactEffect):
            updates_artifact = self.updates_artifact.to_dict()
        else:
            updates_artifact = self.updates_artifact

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "from": from_,
                "id": id,
                "to": to,
            }
        )
        if completes_run is not UNSET:
            field_dict["completesRun"] = completes_run
        if creates_artifact is not UNSET:
            field_dict["createsArtifact"] = creates_artifact
        if creates_gates is not UNSET:
            field_dict["createsGates"] = creates_gates
        if creates_run is not UNSET:
            field_dict["createsRun"] = creates_run
        if requires is not UNSET:
            field_dict["requires"] = requires
        if requires_human is not UNSET:
            field_dict["requiresHuman"] = requires_human
        if requires_passing_blocking_gates is not UNSET:
            field_dict["requiresPassingBlockingGates"] = requires_passing_blocking_gates
        if resumes_run is not UNSET:
            field_dict["resumesRun"] = resumes_run
        if side_stage is not UNSET:
            field_dict["sideStage"] = side_stage
        if updates_artifact is not UNSET:
            field_dict["updatesArtifact"] = updates_artifact

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.workflow_artifact_effect import WorkflowArtifactEffect
        from ..models.workflow_artifact_requirement import WorkflowArtifactRequirement
        from ..models.workflow_run_effect import WorkflowRunEffect

        d = dict(src_dict)

        def _parse_from_(data: object) -> list[str] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                from_type_0 = cast(list[str], data)

                return from_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[str] | None, data)

        from_ = _parse_from_(d.pop("from"))

        id = d.pop("id")

        to = d.pop("to")

        completes_run = d.pop("completesRun", UNSET)

        def _parse_creates_artifact(
            data: object,
        ) -> None | Unset | WorkflowArtifactEffect:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                creates_artifact_type_1 = WorkflowArtifactEffect.from_dict(data)

                return creates_artifact_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | Unset | WorkflowArtifactEffect, data)

        creates_artifact = _parse_creates_artifact(d.pop("createsArtifact", UNSET))

        creates_gates = cast(list[str], d.pop("createsGates", UNSET))

        def _parse_creates_run(data: object) -> None | Unset | WorkflowRunEffect:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                creates_run_type_1 = WorkflowRunEffect.from_dict(data)

                return creates_run_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | Unset | WorkflowRunEffect, data)

        creates_run = _parse_creates_run(d.pop("createsRun", UNSET))

        _requires = d.pop("requires", UNSET)
        requires: list[WorkflowArtifactRequirement] | Unset = UNSET
        if _requires is not UNSET:
            requires = []
            for requires_item_data in _requires:
                requires_item = WorkflowArtifactRequirement.from_dict(
                    requires_item_data
                )

                requires.append(requires_item)

        requires_human = d.pop("requiresHuman", UNSET)

        requires_passing_blocking_gates = d.pop("requiresPassingBlockingGates", UNSET)

        resumes_run = d.pop("resumesRun", UNSET)

        side_stage = d.pop("sideStage", UNSET)

        def _parse_updates_artifact(
            data: object,
        ) -> None | Unset | WorkflowArtifactEffect:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                updates_artifact_type_1 = WorkflowArtifactEffect.from_dict(data)

                return updates_artifact_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | Unset | WorkflowArtifactEffect, data)

        updates_artifact = _parse_updates_artifact(d.pop("updatesArtifact", UNSET))

        workflow_action_definition = cls(
            from_=from_,
            id=id,
            to=to,
            completes_run=completes_run,
            creates_artifact=creates_artifact,
            creates_gates=creates_gates,
            creates_run=creates_run,
            requires=requires,
            requires_human=requires_human,
            requires_passing_blocking_gates=requires_passing_blocking_gates,
            resumes_run=resumes_run,
            side_stage=side_stage,
            updates_artifact=updates_artifact,
        )

        workflow_action_definition.additional_properties = d
        return workflow_action_definition

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
