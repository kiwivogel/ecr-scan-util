library 'jenkins-shared-library@master'

pipeline_component(
    pipeline_libraries: [
        [library: lib_custom_build_script, config: [build_tool: 'sbt']]
    ],

    project_config: [
        name: 'ecr-scan-util',

        deployables: [
            _: 'ecr-scan-util'
        ]
    ]
)