import clsx from 'clsx';
import Heading from '@theme/Heading';
import styles from './styles.module.css';

type FeatureItem = {
  title: string;
  Svg: React.ComponentType<React.ComponentProps<'svg'>>;
  description: JSX.Element;
};

const FeatureList: FeatureItem[] = [
  {
    title: 'Easy to Use',
    Svg: require('@site/static/img/timer.svg').default,
    description: (
      <>
        Helmper was designed from the ground up to be easy to install,
        simple to use and robust enough to rely on.
      </>
    ),
  },
  {
    title: 'Built with CNCF projects',
    Svg: require('@site/static/img/puzzle.svg').default,
    description: (
      <>
        Helmper is built using the same tools you would use on the command line.
        Helmper is using the Helm Go library. For OCI artifact 
        distribution Helmper relies on Oras Go library <code>oras-go</code>.
        Trivy is used  for vulnerability detection and Copacetic is used for vulnerability patching.
      </>
    ),
  },
  {
    title: 'Compatible with most registries',
    Svg: require('@site/static/img/api.svg').default,
    description: (
      <>
        Helmper uses Oras for registry interaction with has support for most registries.
        ACR, ECR, GCR, Harbor, Distribution. See more in the docs.
      </>
    ),
  },
  // {
  //   title: 'No AI',
  //   Svg: require('@site/static/img/undraw_docusaurus_react.svg').default,
  //   description: (
  //     <>
  //       No magic. Helmper relies on boring parsing of Helm Charts and rest/gRPC 
  //       calls to services, so unfortunately Helmper is not using artificial intelligence 
  //       (yet) .
  //     </>
  //   ),
  // },
];

function Feature({title, Svg, description}: FeatureItem) {
  return (
    <div className={clsx('col col--4')}>
      <div className="text--center">
        <Svg className={styles.featureSvg} role="img" />
      </div>
      <div className="text--center padding-horiz--md">
        <Heading as="h3">{title}</Heading>
        <p>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures(): JSX.Element {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
